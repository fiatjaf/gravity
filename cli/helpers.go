package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gogo/protobuf/proto"
	"github.com/mitchellh/go-homedir"
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

func getPrivateKey(name string) (sk *rsa.PrivateKey, err error) {
	dir := getIPFSDir()

	files, err := ioutil.ReadDir(filepath.Join(dir, "keystore"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to list files on keystore: %s", err)
		return
	}

	for _, file := range files {
		if file.Name() == name {
			goto gotkey
		}
	}

	// we don't have a key, create it first
	err = exec.Command("ipfs", "key", "gen", "--type=rsa", "-size=2048", name).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to run 'ipfs key gen': %s", err)
		return
	}

gotkey:
	// read key bytes from file
	data, err := ioutil.ReadFile(filepath.Join(dir, "keystore", name))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to read key file: %s", err)
		return
	}

	pk := new(PrivateKey)
	err = proto.Unmarshal(data, pk)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to unmarshal protobuf data: %s", err)
		return
	}

	sk, err = x509.ParsePKCS1PrivateKey(pk.GetData())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse key: %s", err)
		return
	}

	err = sk.Validate()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Key validation failed: %s", err)
		return
	}

	return
}

func makeJWT(key *rsa.PrivateKey, claims jwt.MapClaims) (token string, err error) {
	return jwt.NewWithClaims(&jwt.SigningMethodRSA{
		Name: "SHA256",
		Hash: crypto.SHA256,
	}, claims).SignedString(key)
}

func printRecord(w io.Writer, value gjson.Result) {
	fmt.Fprintln(w, strings.Join([]string{
		value.Get("owner").String() + "/" + value.Get("name").String(),
		value.Get("cid").String(),
		value.Get("note").String(),
	}, "\t"))
}
