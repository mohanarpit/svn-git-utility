package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	cmap "github.com/streamrail/concurrent-map"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Fatalln("Mandatory arguments missing.")
	}

	command := args[1]
	switch command {
	case "authors":
		var (
			authorFileName = flag.String("filename", "authors.txt",
				"The filename to which the authors will be written into")
			domain = flag.String("domain", "foo.com",
				"The domain for the email addresses of authors")
			svnRepo = flag.String("repo", "/home/svn/repo",
				"The location of the SVN repo on the local FS")
		)
		flag.Parse()

		err := printSvnAuthors(*authorFileName, *domain, *svnRepo)
		if err != nil {
			log.Fatalf("error while fetching authors: %v", err)
		}
	default:
		log.Fatalln("Invalid command")
	}
}

func printSvnAuthors(authorFileName string, domain string, svnRepo string) error {
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
