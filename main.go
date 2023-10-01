package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	flag "github.com/spf13/pflag"
	"github.com/urfave/cli"
)

type LaTeXPreviewFS struct {
	rootDir string
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (ltpfs *LaTeXPreviewFS) Open(name string) (fs.File, error) {
	path := filepath.Join(ltpfs.rootDir, name)
	if filepath.Ext(path) == ".pdf" {
		return os.Open(path)
	}

	dir := filepath.Dir(path)
	dirname := filepath.Base(dir)
	pdfpath := filepath.Join(dir, dirname+".pdf")

	if PathExists(pdfpath) {
		return os.Open(pdfpath)
	}

	pdfpath = filepath.Join(dir, "book.pdf")
	if PathExists(pdfpath) {
		return os.Open(pdfpath)
	}
	return nil, &fs.PathError{
		Op:   "Open",
		Path: name,
		Err:  fs.ErrNotExist,
	}
}

var (
	rootDirF = flag.String("root", "", "root dir of project")
)

func build(path string, lineNumber string, contentFilename string) error {
	dir := filepath.Dir(path)
	dirname := filepath.Base(dir)
	filename := filepath.Base(path)

	fmt.Printf("invoking latexmk in working directory %s with argument %s\n", dir, filename)

	cmd := exec.Command("latexmk", "-synctex=1", "-interaction=nonstopmode", "-pdf", filename)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	pdfFilename := dirname + ".pdf"
	pdfpath := filepath.Join(dir, pdfFilename)

	if !PathExists(pdfpath) {
		pdfFilename = "book.pdf"
	}

	fmt.Printf("invoking displayline in working directory %s with arguments %s %s %s\n", dir, lineNumber, pdfFilename, contentFilename)

	cmd = exec.Command("/Applications/Skim.app/Contents/SharedSupport/displayline", "-r", lineNumber, pdfFilename, contentFilename)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	app := cli.NewApp()
	app.Name = "latexnova"

	app.Commands = []cli.Command{
		{
			Name:    "server",
			Aliases: []string{"s"},
			Usage:   "run http server for latex pdf previews",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "port",
					Usage: "port to listen on; if 0, a random available port will be used",
					Value: "5432",
				},
			},
			Action: func(c *cli.Context) error {
				rd := *rootDirF
				if rd == "" {
					wdp, err := os.Getwd()
					if err != nil {
						log.Fatal(err)
					}
					rd = wdp
				}
				ltpfs := &LaTeXPreviewFS{
					rootDir: rd,
				}

				handler := http.FileServer(http.FS(ltpfs))
				http.Handle("/", handler)

				addr := ":" + c.String("port")
				log.Println("Serving static files at " + addr)
				return http.ListenAndServe(addr, handler)
			},
		},
		{
			Name:    "build",
			Aliases: []string{"b"},
			Usage:   "invoke latexmk",
			Action: func(c *cli.Context) error {
				lineNumber := c.Args().Get(1)
				path := c.Args().First()
				if path == "" {
					return nil
				}
				dir := filepath.Dir(path)
				dirname := filepath.Base(dir)
				texpath := filepath.Join(dir, dirname+".tex")
				contentFilename := filepath.Base(path)

				if PathExists(texpath) {
					return build(texpath, lineNumber, contentFilename)
				}

				texpath = filepath.Join(dir, "main.tex")
				if PathExists(texpath) {
					return build(texpath, lineNumber, contentFilename)
				}
				texpath = filepath.Join(dir, "book.tex")
				if PathExists(texpath) {
					return build(texpath, lineNumber, contentFilename)
				}
				return nil
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
