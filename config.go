package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/yfzhou0904/go-to-kindle/util"
)

type Config struct {
	Email       ConfigEmail       `toml:"email"`
	ScrapingBee ConfigScrapingBee `toml:"scrapingbee"`
}
type ConfigEmail struct {
	SMTPServer string `toml:"smtp_server"`
	Port       int
	From       string
	Password   string
	To         string
}
type ConfigScrapingBee struct {
	APIKey string `toml:"api_key"`
}

func loadConfig() error {
	filepath := filepath.Join(util.BaseDir(), "config.toml")

	// init example config file if does not exist
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		if err = initConfig(filepath); err != nil {
			return err
		}
		if err = openTextEditor(filepath); err != nil {
			return err
		}
	}

	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	return toml.Unmarshal(data, &Conf)
}

func initConfig(path string) error {
	fmt.Println("Initializing example config file at", path)

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(&Conf); err != nil {
		return err
	}

	return nil
}

func openTextEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
