package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	cmap "github.com/streamrail/concurrent-map"
	cli "github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "svn-git-utility"
	app.Usage = "Migrate from svn to git"
	app.Commands = []cli.Command{
		cli.Command{
			Name:        "authors",
			Description: "Will print the list of authors to a filename",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "filename",
					Value: "authorstxt",
					Usage: "The filename to which the authors will be written into",
				},
				cli.StringFlag{
					Name:  "domain",
					Value: "foo.com",
					Usage: "The domain for the email addresses of authors",
				},
				cli.StringFlag{
					Name:  "repo",
					Value: "/home/svn/",
					Usage: "The location of the SVN repo on the local FS",
				},
			},
			Action: func(c *cli.Context) error {
				fmt.Println(c.String("filename"))
				return printSvnAuthors(c.String("filename"), c.String("domain"), c.String("repo"))
			},
		},

		cli.Command{
			Name:        "verify",
			Description: "Will verify if the correct versions of tools are installed",
			Action: func(c *cli.Context) error {
				fmt.Println(c.String("filename"))
				return printSvnAuthors(c.String("filename"), c.String("domain"), c.String("repo"))
			},
		},
	}

	app.Run(os.Args)
}

func printSvnAuthors(authorFileName, domain, svnRepo string) error {
	cmd := exec.Command("svn", "log", "--quiet")
	cmd.Dir = svnRepo
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	outStream := bufio.NewScanner(stdout)

	if err = cmd.Start(); err != nil {
		return err
	}
	//Create the output file
	var file *os.File
	file, err = os.Create(authorFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	authorMap := cmap.New()
	line := make(chan string)
	go writeAuthor(line, file, authorMap, domain)

	go func() {
		for outStream.Scan() {
			line <- outStream.Text()
		}
	}()

	if err = cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func writeAuthor(str chan string, file *os.File, authorMap cmap.ConcurrentMap, domain string) {
	for line := range str {
		if strings.HasPrefix(line, "r") {
			svnString := strings.Split(line, "|")
			author := strings.Trim(svnString[1], " ")
			ok := authorMap.SetIfAbsent(author, true)
			if ok {
				file.Write([]byte(fmt.Sprintf("%s = %s <%s@%s>\n", author, author, author, domain)))
			}
		}
	}
}
