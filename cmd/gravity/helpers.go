package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gogo/protobuf/proto"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
)

func getIPFSDir() string {
	ipfspath := os.Getenv("IPFS_PATH")

	if ipfspath == "" {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		ipfspath = filepath.Join(home, ".ipfs")
	}

	return ipfspath
}

func getPrivateKey() (sk *rsa.PrivateKey, err error) {
	dir := getIPFSDir()

	files, err := ioutil.ReadDir(filepath.Join(dir, "keystore"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to list files on keystore: "+err.Error())
		return
	}

	for _, file := range files {
		if file.Name() == "gravity" {
			goto gotkey
		}
	}

	// we don't have a key, create it first
	err = exec.Command("ipfs", "key", "gen", "--type=rsa", "-size=2048", "gravity").Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to run 'ipfs key gen': "+err.Error())
		return
	}

gotkey:
	// read key bytes from file
	data, err := ioutil.ReadFile(filepath.Join(dir, "keystore", "gravity"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to read key file: "+err.Error())
		return
	}

	pk := new(PrivateKey)
	err = proto.Unmarshal(data, pk)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to unmarshal protobuf data: "+err.Error())
		return
	}

	sk, err = x509.ParsePKCS1PrivateKey(pk.GetData())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse key: "+err.Error())
		return
	}

	err = sk.Validate()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Key validation failed: "+err.Error())
		return
	}

	return
}

func makeJWT(key *rsa.PrivateKey, claims jwt.MapClaims) (token string, err error) {
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
}

func printRecord(w io.Writer, value gjson.Result, quiet bool) {
	if quiet {
		fmt.Fprintln(w, value.Get("cid").String())
		return
	}

	fmt.Fprintln(w, strings.Join([]string{
		value.Get("owner").String() + "/" + value.Get("name").String(),
		value.Get("cid").String(),
		value.Get("note").String(),
	}, "\t"))
}

func printVersion(w io.Writer, t int, value gjson.Result) {
	nseq := strconv.Itoa(t)
	if t == 0 {
		nseq = " 0"
	}

	fmt.Fprintln(w, strings.Join([]string{
		"  ",
		nseq,
		value.Get("date").String(),
		value.Get("cid").String(),
	}, "\t"))
}

func checkCIDExistence(cid string, wait int) bool {
	cmd := exec.Command("ipfs", "object", "stat", cid)

	buf := bytes.NewBuffer([]byte{})
	cmd.Stdout = buf

	err := cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to run 'ipfs object stat': "+err.Error())
		return false
	}

	done := make(chan error)
	go func() { done <- cmd.Wait() }()

	timeout := time.After(time.Duration(wait) * time.Second)

	select {
	case <-timeout:
		cmd.Process.Kill()

		if strings.Index(buf.String(), "Size") != -1 {
			return true
		}

		return false
	case err := <-done:
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error on 'ipfs object stat': "+err.Error())
			return false
		}

		if strings.Index(buf.String(), "Size") != -1 {
			return true
		}
	}

	return false
}

func validateArgKey(cmd *cobra.Command, args []string) error {
	parts := strings.Split(args[0], "/")
	if parts[0] == "" || parts[1] == "" {
		return errors.New("First argument must be in the format <username>/<recordname>.")
	}
	return nil
}

func updateRecord(args []string, key string, value interface{}) {
	sk, err := getPrivateKey()
	if err != nil {
		return
	}

	parts := strings.Split(args[0], "/")
	owner := parts[0]
	name := parts[1]

	// make jwt to send request
	token, err := makeJWT(sk, jwt.MapClaims{
		"owner": owner,
		"name":  name,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to make JWT: "+err.Error())
		return
	}

	req, _ := c.Patch("/"+owner+"/"+name).Set("Token", token).
		BodyJSON(map[string]interface{}{key: value}).Request()
	w, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Request failed: "+err.Error())
		return
	}
	if w.StatusCode >= 300 {
		b, _ := ioutil.ReadAll(w.Body)
		fmt.Fprintln(os.Stderr, string(b))
		return
	}
}

func getCID(key string) string {
	parts := strings.Split(key, "/")
	owner := parts[0]
	name := parts[1]
	req, _ := c.Get("/" + owner + "/" + name).Request()

	w, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Request failed: "+err.Error())
		return ""
	}

	b, _ := ioutil.ReadAll(w.Body)
	j := gjson.ParseBytes(b)

	return j.Get("cid").String()
}
