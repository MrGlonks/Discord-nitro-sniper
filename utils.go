package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/kardianos/osext"

	"github.com/andersfylling/disgord"

	"github.com/bwmarrin/discordgo"
	"github.com/gookit/color"
)

var allSessionsMutex = &sync.Mutex{}

func appendSession(token string, s *disgord.Client) {
	allSessionsMutex.Lock()
	defer allSessionsMutex.Unlock()
	allSessions[token] = s
}

func logWithTime(msg string, endline ...bool) {
	timeStr := time.Now().Format("15:04:05")
	if len(endline) == 0 {
		color.Println("<cyan>" + timeStr + " | </>" + msg)
	} else if !endline[0] {
		color.Print("<cyan>" + timeStr + " | </>" + msg)
	}
}

func fatalWithTime(msg string) {
	timeStr := time.Now().Format("15:04:05")
	color.Println("<cyan>" + timeStr + " | </>" + msg)
	time.Sleep(5 * time.Second)
	os.Exit(1)
}

func deleteEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func loadConfig() ConfigStruct {
	executablePath, err := osext.ExecutableFolder()
	var settings ConfigStruct
	if err != nil {
		log.Fatal("Error: Couldn't determine working directory: " + err.Error())
	}
	os.Chdir(executablePath)
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		fatalWithTime("[x] Failed read file: " + err.Error())
		time.Sleep(4 * time.Second)
		os.Exit(-1)
	}

	err = json.Unmarshal(file, &settings)
	if err != nil {
		fatalWithTime("[x] Failed to parse JSON file: " + err.Error())
		time.Sleep(4 * time.Second)
		os.Exit(-1)
	}
	return settings

}

func removeAltToken(token string) {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	for in, el := range config.AltTokens {
		if el == token {
			//remove token from slice
			config.AltTokens[in] = config.AltTokens[0]
			config.AltTokens = config.AltTokens[1:]
		}
	}
}

func getGuildName(s disgord.Session, m *disgord.MessageCreate) string {
	guild, err := s.GetGuild(m.Ctx, m.Message.GuildID)

	if err != nil || guild == nil {
		return "DM"
	}
	return guild.Name
}

func userIntoUsername(user *disgord.User) string {
	if user == nil || user.Username == "" {
		return ""
	}
	return user.Username + "#" + user.Discriminator.String()
}

func selfIntoUsername(s *discordgo.Session) string {
	user, _ := s.User("@me")
	if user == nil || user.Username == "" {
		return ""
	}
	return user.Username + "#" + user.Discriminator
}

func diffToString(diff time.Duration) string {
	seconds := float64(diff) / float64(time.Second)
	return fmt.Sprintf("%f", seconds)
}


func decrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(text) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)
	data, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func encrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	b := base64.StdEncoding.EncodeToString(text)
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return ciphertext, nil
}
