package main

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/badoux/checkmail"
	"github.com/dghubble/sling"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
)

var server string
var c *sling.Sling

func main() {
	rootCmd.PersistentFlags().
		StringVarP(&server, "server", "s", "ipci.xyz", "IPCI server to use.")
	rootCmd.PersistentFlags().Parse(os.Args[1:])

	baseURL := server
	if !strings.HasPrefix(server, "http") {
		baseURL = "https://" + server
	}
	c = sling.New().Base(baseURL).
		Set("Content-Type", "text/plain").
		Set("Accept", "application/json")

	rootCmd.AddCommand(RegisterCmd)
	rootCmd.AddCommand(PutCmd)
	rootCmd.AddCommand(GetCmd)
	rootCmd.AddCommand(DelCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "ipci",
	Short: "ipci - intraplanetary centralized index",
	Long: `ipci - intraplanetary centralized index

ipci is a centralized index for all files distributed over IPFS.

You can use ipci as a hub to which you can announce data you've made available through IPFS or in which you'll find interesting stuff from others to pin.
    `,
	Version: "v0",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of a ipci",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ipci v0")
	},
}

var RegisterCmd = &cobra.Command{
	Use:   "register [username] [email]",
	Short: "Register with ipci.xyz",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("2 arguments are required, username and email address.")
		}
		return checkmail.ValidateFormat(args[1])
	},
	Run: func(cmd *cobra.Command, args []string) {
		username := args[0]
		email := args[1]

		// get public key pem
		sk, err := getPrivateKey("ipci")
		if err != nil {
			return
		}

		// send everything
		body := &bytes.Buffer{}
		if err = pem.Encode(body, &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(&sk.PublicKey),
		}); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to encode public key: %s", err)
			return
		}

		req, _ := c.Post("/"+username).Set("Email", email).Body(body).Request()
		w, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Request failed: %s", err)
			return
		}
		if w.StatusCode >= 300 {
			b, _ := ioutil.ReadAll(w.Body)
			fmt.Fprintln(os.Stderr, string(b))
			return
		}
	},
}

var GetCmd = &cobra.Command{
	Use:   "get [key or cid]",
	Short: "Get something from ipci.xyz",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var req *http.Request

		if strings.IndexByte(args[0], '/') == -1 {
			cid := args[0]
			req, _ = c.Get("/").QueryStruct(CIDQuery{cid}).Request()
		} else {
			parts := strings.Split(args[0], "/")
			owner := parts[0]
			name := parts[1]
			req, _ = c.Get("/" + owner + "/" + name).Request()
		}

		w, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Request failed: %s", err)
			return
		}

		b, _ := ioutil.ReadAll(w.Body)
		j := gjson.ParseBytes(b)
		tw := tabwriter.NewWriter(os.Stdout, 3, 3, 2, ' ', 0)

		if j.IsArray() {
			j.ForEach(func(_, value gjson.Result) bool {
				printRecord(tw, value)
				return true
			})
		} else if j.IsObject() {
			printRecord(tw, j)
		}

		tw.Flush()
	},
}

var PutCmd = &cobra.Command{
	Use:   "put [key] [ipfs cid]",
	Short: "Put something on ipci.xyz",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return errors.New("2 arguments are required, username/recordname and record hash.")
		}
		parts := strings.Split(args[0], "/")
		if parts[0] == "" || parts[1] == "" {
			return errors.New("First argument must be username/recordname.")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		sk, err := getPrivateKey("ipci")
		if err != nil {
			return
		}

		parts := strings.Split(args[0], "/")
		owner := parts[0]
		name := parts[1]
		cid := args[1]

		token, err := makeJWT(sk, jwt.MapClaims{
			"owner": owner,
			"name":  name,
			"cid":   cid,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to make JWT: %s", err)
			return
		}

		req, _ := c.Put("/"+owner+"/"+name).Set("Token", token).
			Body(bytes.NewBufferString(cid)).Request()
		w, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Request failed: %s", err)
			return
		}
		if w.StatusCode >= 300 {
			b, _ := ioutil.ReadAll(w.Body)
			fmt.Fprintln(os.Stderr, string(b))
			return
		}
	},
}

var DelCmd = &cobra.Command{
	Use:   "del",
	Short: "Delete something from ipci.xyz",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("1 arguments is required: username/recordname.")
		}
		parts := strings.Split(args[0], "/")
		if parts[0] == "" || parts[1] == "" {
			return errors.New("Argument must be username/recordname.")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		sk, err := getPrivateKey("ipci")
		if err != nil {
			return
		}

		parts := strings.Split(args[0], "/")
		owner := parts[0]
		name := parts[1]

		token, err := makeJWT(sk, jwt.MapClaims{
			"owner": owner,
			"name":  name,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to make JWT: %s", err)
			return
		}

		req, _ := c.Delete("/"+owner+"/"+name).Set("Token", token).Request()
		w, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Request failed: %s", err)
			return
		}
		if w.StatusCode >= 300 {
			b, _ := ioutil.ReadAll(w.Body)
			fmt.Fprintln(os.Stderr, string(b))
			return
		}
	},
}
