// GitHub Release 正文渲染与子命令独立存放，避免发布路径与审计路径交织。
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func runRender(arguments []string) error {
	flags := flag.NewFlagSet("render", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	version := flags.String("version", "", "semantic version without v")
	directory := flags.String("dir", "", "version directory containing en.md and zh-CN.md")
	output := flags.String("output", "", "optional output file; stdout when omitted")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 || !semanticVersionPattern.MatchString(*version) || *directory == "" {
		return errors.New("render requires --version and --dir")
	}
	body, err := renderGitHubReleaseBody(*directory, *version)
	if err != nil {
		return err
	}
	if *output == "" {
		_, err = fmt.Fprint(os.Stdout, body)
		return err
	}
	return os.WriteFile(*output, []byte(body), 0o644)
}
