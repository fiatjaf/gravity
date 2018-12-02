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
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/badoux/checkmail"
	"github.com/dghubble/sling"
	"github.com/dgrijalva/jwt-go"
	"github.com/gumieri/open-in-editor"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
)

var server string
var c *sling.Sling
var wait int
var putNote string
var quiet bool
var showVersions bool

func main() {
	rootCmd.PersistentFlags().
		StringVarP(&server, "server", "s", "bigsun.xyz", "Gravity server to use.")
	rootCmd.PersistentFlags().Parse(os.Args[1:])

	GetCmd.Flags().
		BoolVarP(&quiet, "quiet", "Q", false, "Print only final hash.")
	GetCmd.Flags().
		BoolVarP(&showVersions, "history", "H", false, "Show old versions.")
	GetCmd.Flags().Parse(os.Args[1:])

	PutCmd.Flags().
		StringVarP(&putNote, "note", "n", "", "A note to identify this object.")
	PutCmd.Flags().
		IntVarP(&wait, "wait", "w", 2, "Time to wait for 'ipfs object stat'.")
	PutCmd.Flags().Parse(os.Args[1:])

	baseURL := server
	if !strings.HasPrefix(server, "http") {
		baseURL = "https://" + server
	}
	c = sling.New().Base(baseURL).
		Set("Content-Type", "text/plain").
		Set("Accept", "application/json")

	rootCmd.AddCommand(RegisterCmd)
	rootCmd.AddCommand(PutCmd)
	rootCmd.AddCommand(RenameCmd)
	rootCmd.AddCommand(NoteCmd)
	rootCmd.AddCommand(BodyCmd)
	rootCmd.AddCommand(GetCmd)
	rootCmd.AddCommand(StatCmd)
	rootCmd.AddCommand(DelCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "gravity",
	Short: "gravity - intraplanetary centralized index",
	Long: `gravity - intraplanetary centralized index

gravity is a centralized index for all files distributed over IPFS.

You can use gravity as a hub to which you can announce data you've made available through IPFS or in which you'll find interesting stuff from others to pin.
    `,
	Version: "v2",
}

var RegisterCmd = &cobra.Command{
	Use:   "register [username] [email]",
	Short: "Register in the gravity server",
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
		sk, err := getPrivateKey()
		if err != nil {
			return
		}

		// send everything
		body := &bytes.Buffer{}
		if err = pem.Encode(body, &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(&sk.PublicKey),
		}); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to encode public key: "+err.Error())
			return
		}

		req, _ := c.Post("/"+username).Set("Email", email).Body(body).Request()
		w, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Request failed: "+err.Error())
			return
		}
		if w.StatusCode >= 300 {
			b, _ := ioutil.ReadAll(w.Body)
			fmt.Fprint(os.Stderr, string(b))
			return
		}
	},
}

var GetCmd = &cobra.Command{
	Use:        "get [key or cid]",
	Aliases:    []string{"query", "fetch", "list", "ls"},
	SuggestFor: []string{"find"},
	Short:      "Get something from the gravity server",
	Example: `~> gravity get fiatjaf/gravity
fiatjaf/gravity  QmQjyLocqMrwxNnz5G1UtHZrRNsztgR97jLtch7bK28BWa  precompiled binaries for the gravity CLI tool.

~> gravity get fiatjaf/bitcoin.pdf -Q
QmRA3NWM82ZGynMbYzAgYTSXCVM14Wx1RZ8fKP42G6gjgj

~> gravity get fiatjaf/gravity --history
fiatjaf/gravity  QmQjyLocqMrwxNnz5G1UtHZrRNsztgR97jLtch7bK28BWa  precompiled binaries for the gravity CLI tool.                                                                               
     0  2018-11-28 11:39:53.338399  zdj7Wmp2eLDSEQXiFDFaJqkyBdAtjcf84fLWtC3UGC1oW7pUT
    -1  2018-11-16 03:13:32.176537  QmQjyLocqMrwxNnz5G1UtHZrRNsztgR97jLtch7bK28BWa
    -2  2018-11-14 20:34:36.67102   QmVQ3zYTPnnu7iggGh7Cpr9naL7VDZ8x8cWd2EMexDv3w

~> gravity get fiatjaf/
fiatjaf/videos              zdj7Wa7HGxHGfAb1o9xRFDhbSjuWqVBAaavRw5WX4BEVV8YD5  some videos worth saving.
fiatjaf/olavodecarvalho.org zdj7WetgxoFSiPJSKCn9asF77TLh7Kb3eDGpgh4VPJm93zssA  olavodecarvalho.org old website.
fiatjaf/gravity             QmQjyLocqMrwxNnz5G1UtHZrRNsztgR97jLtch7bK28BWa     precompiled binaries for the gravity CLI tool.
fiatjaf/bitcoin.pdf         QmRA3NWM82ZGynMbYzAgYTSXCVM14Wx1RZ8fKP42G6gjgj     
fiatjaf/fiatjaf.alhur.es    QmT5vWxZ1qTePvZg9NJAJDBJtZ81UGu9MoVbsmJoc946ho     my personal website.

~> gravity find QmVQ3zYTPnnu7iggGh7Cpr9naL7VDZ8x8cWd2EMexDv3w
fiatjaf/gravity
    -2  2018-11-14 20:34:36.67102   QmVQ3zYTPnnu7iggGh7Cpr9naL7VDZ8x8cWd2EMexDv3w

~> gravity list fiatjaf/cof/
zdj7WiDR1mUKjC7PU5aVTU3s3Dkd9UW2pQe9UsXPi4WQ62yTt 13680450440 aulas/
zdj7Wf5jP4JgV9HLUjtTJx2mqFSpwWNoK3EtpMT2myUjdBEpo 9690704     transcrições/

~> gravity list fiatjaf/cof/aulas/
zdj7WWfpuLo8qJcM2S3WBx9Coo7Gqy8EzBB77F768E5pX5uc4 13538149  000/
zdj7WjJMeYNUYxTSUFikWiDVmAwFvExzKjBZyvC71hrrx5V6x 36802688  001/
zdj7WbCpyr2qx765pmd5TbeiRmrQdQbnWQPVK6LuGPUfCGksM 53582473  002/
`,
	Run: func(cmd *cobra.Command, args []string) {
		var cidquery bool
		var req *http.Request

		if len(args) == 0 {
			req, _ = c.Get("/").Request()
		} else if strings.IndexByte(args[0], '/') == -1 {
			cid := args[0]
			req, _ = c.Get("/?cid=" + cid).Request()
			cidquery = true
		} else {
			parts := strings.Split(args[0], "/")
			owner := parts[0]
			name := parts[1]
			path := "/" + owner + "/" + name
			if showVersions {
				path += "?full=1"
			}
			req, _ = c.Get(path).Request()
		}

		w, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Request failed: "+err.Error())
			return
		}

		b, _ := ioutil.ReadAll(w.Body)
		j := gjson.ParseBytes(b)
		tw := tabwriter.NewWriter(os.Stdout, 3, 3, 2, ' ', 0)

		if j.IsArray() {
			if !cidquery {
				// it's a list of all the records, or all the records for one user
				j.ForEach(func(_, value gjson.Result) bool {
					printRecord(tw, value, quiet)
					return true
				})
			} else {
				// it's a list of history entries
				tw.Flush()
				j.ForEach(func(_, h gjson.Result) bool {
					fmt.Fprintf(os.Stdout, "%s/%s\n",
						h.Get("owner").String(), h.Get("name").String())
					tw = tabwriter.NewWriter(os.Stdout, 3, 3, 2, ' ', 0)
					printVersion(tw, int(-h.Get("nseq").Int()), h)
					tw.Flush()
					return true
				})
			}
		} else if j.IsObject() {
			// it's just one record
			parts := strings.Split(args[0], "/")

			if len(parts) > 2 {
				// command was called with more than two strings in the path,
				// so we'll call `ipfs ls` on the result
				cid := j.Get("cid").String()
				cmdls := exec.Command("ipfs", "ls", cid+"/"+strings.Join(parts[2:], "/"))
				cmdls.Stderr = os.Stderr
				cmdls.Stdout = os.Stdout
				err := cmdls.Run()
				if err != nil {
					fmt.Fprintln(os.Stderr, "Unable to run 'ipfs ls': "+err.Error())
					return
				}
			} else {
				// just print the record data
				if showVersions {
					quiet = false
				}

				printRecord(tw, j, quiet)

				if showVersions {
					// flush previous and start a new tabwriter for this
					tw.Flush()
					tw = tabwriter.NewWriter(os.Stdout, 3, 3, 2, ' ', 0)

					t := 0
					j.Get("history").ForEach(func(_, h gjson.Result) bool {
						printVersion(tw, t, h)
						t--
						return true
					})
				}
			}
		}

		tw.Flush()
	},
}

var StatCmd = &cobra.Command{
	Use:     "stat [key[/path]]",
	Aliases: []string{"info"},
	Short:   "Get a hash from the gravity server and call `ipfs object stat` on it or in a subpath of it.",
	Args:    cobra.ExactArgs(1),
	Example: `~> gravity stat fiatjaf/bitcoin.pdf
NumLinks: 0
BlockSize: 184306
LinksSize: 4
DataSize: 184302
CumulativeSize: 184306
`,
	Run: func(cmd *cobra.Command, args []string) {
		parts := strings.Split(args[0], "/")
		key := parts[0] + "/" + parts[1]

		cid := getCID(key)
		if cid == "" {
			return
		}

		cmdstat := exec.Command("ipfs", "object", "stat",
			cid+"/"+strings.Join(parts[2:], "/"))
		cmdstat.Stderr = os.Stderr
		cmdstat.Stdout = os.Stdout
		err := cmdstat.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Unable to run 'ipfs object stat': "+err.Error())
			return
		}
	},
}

var PutCmd = &cobra.Command{
	Use:     "put [key] [ipfs cid]",
	Short:   "Put something on the gravity server",
	Args:    validateArgKey,
	Example: `~> gravity put fiatjaf/bitcoin.pdf QmRA3NWM82ZGynMbYzAgYTSXCVM14Wx1RZ8fKP42G6gjgj`,
	Run: func(cmd *cobra.Command, args []string) {
		sk, err := getPrivateKey()
		if err != nil {
			return
		}

		parts := strings.Split(args[0], "/")
		owner := parts[0]
		name := parts[1]
		note := putNote
		cid := args[1]

		// check if we have the file
		ok := checkCIDExistence(cid, wait)
		if !ok {
			fmt.Fprintln(os.Stderr, "File not available on IPFS.")
			return
		}

		// make jwt to send request
		token, err := makeJWT(sk, jwt.MapClaims{
			"owner": owner,
			"name":  name,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to make JWT: "+err.Error())
			return
		}

		req, _ := c.Put("/"+owner+"/"+name).Set("Token", token).
			BodyJSON(map[string]interface{}{"cid": cid, "note": note}).Request()
		w, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Request failed: "+err.Error())
			return
		}
		if w.StatusCode >= 300 {
			b, _ := ioutil.ReadAll(w.Body)
			fmt.Fprint(os.Stderr, string(b))
			return
		}
	},
}

var RenameCmd = &cobra.Command{
	Use:   "rename [key] [name]",
	Short: "Rename a record.",
	Args:  validateArgKey,
	Run: func(cmd *cobra.Command, args []string) {
		updateRecord(args, "name", args[1])
	},
}

var NoteCmd = &cobra.Command{
	Use:   "note [key] [note]",
	Short: "Set a note for the object given by [key].",
	Args:  validateArgKey,
	Run: func(cmd *cobra.Command, args []string) {
		updateRecord(args, "note", args[1])
	},
}

var BodyCmd = &cobra.Command{
	Use:     "body [key]",
	Aliases: []string{"edit"},
	Short:   "Edit the Markdown body for the object given by [key] in your local editor.",
	Args:    validateArgKey,
	Run: func(cmd *cobra.Command, args []string) {
		program := os.Getenv("EDITOR")
		if program == "" {
			program = "editor"
		}
		editor := openineditor.Editor{Command: program}

		// fetch current contents
		parts := strings.Split(args[0], "/")
		owner := parts[0]
		name := parts[1]
		req, _ := c.Get("/" + owner + "/" + name + "?full=1").Request()
		w, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Request failed: "+err.Error())
			return
		}
		b, _ := ioutil.ReadAll(w.Body)
		contents := gjson.GetBytes(b, "body").String()
		if contents == "" {
			contents += `## ` + name + `

An amazing thing.
`
		}

		f := &openineditor.File{FileName: name, Content: []byte(contents)}
		err = f.CreateInTempDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to create temporary file: "+err.Error())
			return
		}

		err = editor.OpenTempFile(f)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to open file for editing: "+err.Error())
			return
		}

		updateRecord(args, "body", string(f.Content))
	},
}

var DelCmd = &cobra.Command{
	Use:   "del [key]",
	Short: "Delete something from the gravity server",
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
		sk, err := getPrivateKey()
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
			fmt.Fprintln(os.Stderr, "Failed to make JWT: "+err.Error())
			return
		}

		req, _ := c.Delete("/"+owner+"/"+name).Set("Token", token).Request()
		w, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Request failed: "+err.Error())
			return
		}
		if w.StatusCode >= 300 {
			b, _ := ioutil.ReadAll(w.Body)
			fmt.Fprint(os.Stderr, string(b))
			return
		}
	},
}
