package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const socketPath = "/tmp/mpvsocket"

var FileName string
var AlbumName string

func sendToMPV(cmd []interface{}) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Fatalf("Erro conectando ao mpv: %v", err)
	}
	defer conn.Close()

	msg := map[string]interface{}{"command": cmd}
	if err := json.NewEncoder(conn).Encode(msg); err != nil {
		log.Fatalf("Erro enviando comando: %v", err)
	}
}

func startMPV(albumPath string) {
	fileStats, _ := os.Stat(albumPath)
	if !fileStats.IsDir() {
		args := append([]string{"--no-video", "--input-ipc-server=" + socketPath}, albumPath)
		cmd := exec.Command("mpv", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Start()
		return
	}
	rawFilesName, err := os.ReadDir(albumPath)
	if err != nil || len(rawFilesName) == 0 {
		log.Fatalf("Nenhuma música encontrada em: %s", albumPath)
	}

	var files []string
	for _, v := range rawFilesName {
		if !v.IsDir() && (strings.HasSuffix(v.Name(), ".mp3") || strings.HasSuffix(v.Name(), ".flac") || strings.HasSuffix(v.Name(), ".wav")) {
			files = append(files, filepath.Join(albumPath, v.Name()))
		}
	}

	args := append([]string{"--no-video", "--input-ipc-server=" + socketPath}, files...)
	cmd := exec.Command("mpv", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
}

func completeArtists(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	musicDir := filepath.Join(os.Getenv("HOME"), "Músicas")
	var results []string

	filepath.Walk(musicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && path != musicDir {
			rel, _ := filepath.Rel(musicDir, path)
			results = append(results, rel)
			return filepath.SkipDir
		}
		return nil
	})
	return results, cobra.ShellCompDirectiveNoFileComp
}

func completeAudioFiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	artist := strings.Join(args, " ")
	artist = strings.TrimSpace(artist)
	if strings.HasPrefix(toComplete, "-") {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	musicDir := filepath.Join(os.Getenv("HOME"), "Músicas", artist, AlbumName)
	debug := fmt.Sprintf("ARGS: %q\nTO_COMPLETE: %q\nPATH: %q\n", args, toComplete, musicDir)
	os.WriteFile("/tmp/debug.txt", []byte(debug), 0644)

	var results []string

	dirEntries, err := os.ReadDir(musicDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	for _, entry := range dirEntries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".mp3") || strings.HasSuffix(entry.Name(), ".flac") || strings.HasSuffix(entry.Name(), ".wav")) {
			results = append(results, entry.Name())
		}
	}
	return results, cobra.ShellCompDirectiveNoFileComp
}
func completeAlbunsFolder(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	artista := strings.Join(args, " ")
	artista = strings.TrimSpace(artista)
	if strings.HasPrefix(toComplete, "-") {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	musicDir := filepath.Join(os.Getenv("HOME"), "Músicas", artista)
	var results []string
	filepath.Walk(musicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && path != musicDir {
			rel, _ := filepath.Rel(musicDir, path)
			results = append(results, rel)
			return filepath.SkipDir
		}
		return nil
	})
	return results, cobra.ShellCompDirectiveNoFileComp
}

// Comandos
var rootCmd = &cobra.Command{
	Use:   "bard",
	Short: "O bardo musical no terminal",
	Long:  "Um utilitário para tocar música usando o MPV com controle interativo via IPC.",
}

var playCmd = &cobra.Command{
	Use:               "play [artista]",
	Short:             "Toca todas as músicas de uma pasta",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeArtists,
	Run: func(cmd *cobra.Command, args []string) {
		album := args[0]
		path := filepath.Join(os.Getenv("HOME"), "Músicas", album)
		if AlbumName != "" {
			path += "/" + AlbumName

		}
		if FileName != "" {
			path += "/" + FileName
		}
		startMPV(path)
	},
}

var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pausa ou continua a reprodução",
	Run: func(cmd *cobra.Command, args []string) {
		sendToMPV([]interface{}{"cycle", "pause"})
	},
}
var killCmd = &cobra.Command{
	Use:   "exit",
	Short: "Mata todas as execuções do bard",
	Run: func(c *cobra.Command, args []string) {
		fmt.Println("Matando todas as instâncias do mpv...")
		proc := exec.Command("pkill", "-f", "mpv")
		output, err := proc.CombinedOutput()
		if err != nil {
			log.Fatalf("Erro ao executar pkill: %v\nSaída: %s", err, string(output))
		}
	},
}

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Avança para a próxima música",
	Run: func(cmd *cobra.Command, args []string) {
		sendToMPV([]interface{}{"playlist-next"})
	},
}
var prevCmd = &cobra.Command{
	Use:   "prev",
	Short: "Volta para a música anterior",
	Run: func(cmd *cobra.Command, args []string) {
		sendToMPV([]interface{}{"playlist-prev"})
	},
}

var volCmd = &cobra.Command{
	Use:   "vol [0-100]",
	Short: "Ajusta o volume",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sendToMPV([]interface{}{"set_property", "volume", args[0]})
	},
}

func init() {
	rootCmd.AddCommand(playCmd)
	playCmd.Flags().StringVarP(&AlbumName, "album", "a", "", "Album name (Required)")
	playCmd.RegisterFlagCompletionFunc("album", completeAlbunsFolder)
	playCmd.MarkFlagRequired("album")
	playCmd.Flags().StringVarP(&FileName, "music", "m", "", "File name (optional)")
	playCmd.RegisterFlagCompletionFunc("music", completeAudioFiles)
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(killCmd)
	rootCmd.AddCommand(prevCmd)
	rootCmd.AddCommand(volCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
