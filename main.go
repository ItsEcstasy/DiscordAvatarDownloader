package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type Settings struct {
	Token     string   `json:"token"`
	ServerIDs []string `json:"serverIDs"`
	OutputDir string   `json:"outputDir"`
}

func download(url, out string, wg *sync.WaitGroup) {
	defer wg.Done()

	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		// fmt.Printf("Failed | Link: %s\n", url)
		return
	}
	defer resp.Body.Close()

	avatarFile := filepath.Base(url)
	if strings.Contains(avatarFile, "?") {
		parts := strings.Split(avatarFile, "?")
		avatarFile = parts[0]
	}

	avatarFile = strings.Replace(avatarFile, ".webp", ".png", 1)

	serverDir := filepath.Join(out, sanitize(filepath.Base(filepath.Dir(out))))
	filePath := filepath.Join(serverDir, avatarFile)

	if err := os.MkdirAll(serverDir, os.ModePerm); err != nil {
		fmt.Printf("Error creating directory for server: %s\n", err)
		return
	}

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Failed to create file: %s\n", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		fmt.Printf("Failed to save file: %s\n", err)
		return
	}

	fmt.Printf("Success | Link: %s\n", url)
}

func sanitize(name string) string {
	invalidChars := regexp.MustCompile(`[:*?"<>|]`)
	return invalidChars.ReplaceAllString(name, "_")
}

func main() {
	file, err := os.ReadFile("settings.json")
	if err != nil {
		fmt.Println("Error reading settings file:", err)
		return
	}

	var settings Settings
	err = json.Unmarshal(file, &settings)
	if err != nil {
		fmt.Println("Error parsing settings:", err)
		return
	}

	if settings.Token == "" || settings.ServerIDs == nil || len(settings.ServerIDs) == 0 {
		fmt.Println("settings.json is not properly setup.")
		return
	}

	baseDir := filepath.Join(".", settings.OutputDir)
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		fmt.Printf("Error creating base path: %s\n", err)
		return
	}

	dg, err := discordgo.New("Bot " + settings.Token)
	if err != nil {
		fmt.Println("Error creating session:", err)
		return
	}
	defer dg.Close()

	for _, serverID := range settings.ServerIDs {
		guild, err := dg.Guild(serverID)
		if err != nil {
			fmt.Println("Error fetching server info:", err)
			continue
		}

		outputDir := filepath.Join(baseDir, sanitize(guild.Name))

		err = os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			fmt.Printf("Error creating path for %s: %s\n", guild.Name, err)
			continue
		}

		members, err := dg.GuildMembers(serverID, "", 1000)
		if err != nil {
			fmt.Println("Error fetching members:", err)
			continue
		}

		var wg sync.WaitGroup
		for _, member := range members {
			if member.User.Avatar != "" {
				avatarURL := member.User.AvatarURL("512") // 512 x 512
				wg.Add(1)
				go download(avatarURL, outputDir, &wg)
			}
		}

		wg.Wait()
		fmt.Printf("Finished downloading avatars for %s.\n", guild.Name)
	}
}
