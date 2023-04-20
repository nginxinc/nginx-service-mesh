// Package main the main entry point for the cli
package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/nginxinc/nginx-service-mesh/internal/nginx-meshctl/commands"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	pkgName = "nginx-meshctl"
	version = "0.0.0"
	commit  = "local"
	genDocs = ""
)

func main() {
	cmd := commands.Setup(pkgName, version, commit)
	cmd.SilenceUsage = true

	if genDocs != "" {
		if err := generateMarkdown(cmd); err != nil {
			log.Fatal(err)
		}
	}

	if cmd.Execute() != nil {
		os.Exit(1)
	}
}

func generateCompletion(cmd *cobra.Command, docsPath string) error {
	cmd.CompletionOptions.HiddenDefaultCmd = false
	cmd.InitDefaultCompletionCmd()
	completionCmd, _, err := cmd.Find([]string{"completion"})
	if err != nil {
		return err
	}
	mdFileFullPath := filepath.Join(docsPath, "nginx-meshctl_completion.md")

	mdFile, err := os.Create(mdFileFullPath)
	if err != nil {
		return err
	}
	defer mdFile.Close()

	caser := cases.Title(language.English)

	scanner := bufio.NewScanner(strings.NewReader(completionCmd.UsageString()))
	if _, err = mdFile.WriteString("## " + caser.String(completionCmd.Name()) + "\n"); err != nil {
		return err
	}

	for scanner.Scan() {
		line := strings.ReplaceAll(scanner.Text(), "ke.agarwal", "<user>")
		if strings.Contains(line, "Usage:") {
			_, err = mdFile.WriteString("```\n")
		} else if strings.Contains(line, "Available Commands:") {
			_, err = mdFile.WriteString("```\n### " + line + "\n```\n")
		} else if strings.Contains(line, "Global Flags:") {
			_, err = mdFile.WriteString("```\n### Options\n```\n")
		} else if strings.Contains(line, "for more information about") {
			_, err = mdFile.WriteString("```\n")
		} else if len(line) > 0 {
			_, err = mdFile.WriteString(line + "\n")
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func generateMarkdown(cmd *cobra.Command) error {
	cmd.DisableAutoGenTag = true

	// generates a markdown file for each CLI command
	_, b, _, _ := runtime.Caller(0)
	docsPath := filepath.Join(filepath.Dir(b), "../../docs/content/reference")
	if err := doc.GenMarkdownTree(cmd, docsPath); err != nil {
		return err
	}

	// combine all files in a buffer
	buf := new(strings.Builder)
	if err := generateCompletion(cmd, docsPath); err != nil {
		log.Fatal(err)
	}

	docsFiles, err := os.ReadDir(docsPath)
	if err != nil {
		return err
	}

	for _, file := range docsFiles {
		if file.IsDir() || !strings.HasPrefix(file.Name(), "nginx-meshctl") {
			continue
		}

		mdFileFullPath := filepath.Join(docsPath, file.Name())

		var mdFile *os.File
		mdFile, err = os.Open(mdFileFullPath)
		if err != nil {
			return err
		}
		defer mdFile.Close()

		_, err = io.Copy(buf, mdFile)
		if err != nil {
			return err
		}
		if _, err = buf.WriteString("\n"); err != nil {
			return err
		}

		if err = os.Remove(mdFileFullPath); err != nil {
			return err
		}
	}

	finalName := filepath.Join(docsPath, "nginx-meshctl.md")
	finalFile, err := os.Create(finalName)
	if err != nil {
		return err
	}
	defer finalFile.Close()

	if _, err := finalFile.WriteString(`---
	title: CLI Reference
	description: "Man page and instructions for using the NGINX Service Mesh CLI"
	draft: false
	weight: 300
	toc: true
	categories: ["reference"]
	docs: "DOCS-704"
	---
`); err != nil {
		return err
	}

	re := regexp.MustCompile("## nginx-meshctl ([a-z]+)")
	caser := cases.Title(language.English)
	scanner := bufio.NewScanner(strings.NewReader(buf.String()))
	for scanner.Scan() {
		line := strings.ReplaceAll(scanner.Text(), "ke.agarwal", "<user>")
		if list := re.FindStringSubmatch(line); list != nil {
			if _, err := finalFile.WriteString("## " + caser.String(list[len(list)-1]) + "\n"); err != nil {
				return err
			}
			continue
		}
		// don't add cobra-generated footer
		if !strings.Contains(line, "SEE ALSO") && !strings.Contains(line, "* [nginx-meshctl") {
			if _, err := finalFile.WriteString(line + "\n"); err != nil {
				return err
			}
		}
	}

	return nil
}
