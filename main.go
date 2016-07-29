package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

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
				return printAuthorsCommand(c.String("filename"), c.String("domain"), c.String("repo"))
			},
		},

		cli.Command{
			Name:        "verify",
			Description: "Will verify if the correct versions of tools are installed",
			Action: func(c *cli.Context) error {
				return verifyCommand()
			},
		},
	}
	app.Run(os.Args)
}

type Dependency struct {
	Name            string
	RequiredVersion string
	Cmd             string
}

type DependencyOutput struct {
	Dependency
	Output string
}

func verifyCommand() error {
	tools := []Dependency{
		{
			Name:            "Git",
			RequiredVersion: "1.7.7.5",
			Cmd:             "git",
		},
		{
			Name:            "svn",
			RequiredVersion: "1.6.17",
			Cmd:             "svn",
		},
		{
			Name:            "git-svn",
			RequiredVersion: "1.7.7.5",
			Cmd:             "git svn",
		},
	}

	var wg sync.WaitGroup
	wg.Add(len(tools) + 1)

	errChan := make(chan error)
	go func() {
		for {
			select {
			case err := <-errChan:
				if err != nil {
					fmt.Println(err)
				}
				wg.Done()
			}
		}
	}()

	go checkConnectivity(errChan)
	for _, tool := range tools {
		go verify(tool, errChan)
	}
	wg.Wait()
	return nil
}

func checkConnectivity(errChan chan error) {
	conn, err := net.Dial("tcp", "google.com:80")
	if err != nil {
		errChan <- fmt.Errorf("Unable to connect to the internet. Please check your connectivity")
		return
	}
	defer conn.Close()
}

func verify(dep Dependency, errChan chan error) {
	var stdout []byte
	var err error
	//Explicitly making this distinction because exec.Command doesn't work when there's a space separated string
	//It assumes that the space separated string is an entire command and not a command + argument
	//TODO: Figure out how this can be cleaned up
	if dep.Name == "git-svn" {
		stdout, err = exec.Command("git", "svn", "--version").Output()
	} else {
		stdout, err = exec.Command(dep.Cmd, "--version").Output()
	}

	if err != nil {
		errChan <- err
		return
	}
	//Get the first line
	firstLine := bytes.Split(stdout, []byte("\n"))[0]
	re := regexp.MustCompile(`version ([0-9.]+)`)
	match := re.FindStringSubmatch(string(firstLine))
	version := match[1]
	if version < dep.RequiredVersion {
		errChan <- fmt.Errorf("Sorry, the installed version for %s is less than the required version %s", dep.Name, dep.RequiredVersion)
		return
	}
	errChan <- nil
}

func printAuthorsCommand(authorFileName, domain, svnRepo string) error {
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
