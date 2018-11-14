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
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gogo/protobuf/proto"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
)

type CIDQuery struct {
	CID string `url:"cid"`
}

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

func printRecord(w io.Writer, value gjson.Result) {
	fmt.Fprintln(w, strings.Join([]string{
		value.Get("owner").String() + "/" + value.Get("name").String(),
		value.Get("cid").String(),
		value.Get("note").String(),
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

func validateArgsRecord(second string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("2 arguments are required, username/recordname and " + second + ".")
		}
		parts := strings.Split(args[0], "/")
		if parts[0] == "" || parts[1] == "" {
			return errors.New("First argument must be username/recordname.")
		}
		return nil
	}
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
