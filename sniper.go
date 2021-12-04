package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	strconv "strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/toast.v1"

	"github.com/andersfylling/disgord"
	"github.com/valyala/fasthttp"
	"golang.org/x/sync/syncmap"
)

type ConfigStruct struct {
	MainToken           string
	AltTokens           []string
	webhookNitroClaimed string
	NitroClaimedLimit   int
}

var (
	config            ConfigStruct
	paymentSourceIDs  = make(map[string]string)
	reGiftLink        = regexp.MustCompile("(discord.com/gifts/|discordapp.com/gifts/|discord.gift/)([a-zA-Z0-9]+)")
	rePaymentSourceID = regexp.MustCompile(`("id": ")([0-9]+)"`)
	tokenMutex        = &sync.Mutex{}
	allSessions       = make(map[string]*disgord.Client)
	mainUsername      string
	mainAvatar        string
	mainClient        *disgord.Client
	nitroClaimed      = 0
	nitroFailed       = 0
	tokens            = syncmap.Map{}
	mainID            disgord.Snowflake
	VERSION           = "v0.1"
	Sniping           = true
	mainFinished      = make(chan bool)
	AltsIds           = []uint{}
	username          = ""
	password          = ""
)


func main() {

	os.Stderr = nil
	c := exec.Command("cmd", "/c", "cls")
	c.Stdout = os.Stdout
	c.Run()

	go func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGKILL)
		<-sc

		os.Exit(0)
	}()
	config = loadConfig()

	setupFinished := make(chan bool)

	if config.MainToken == "" {
		fatalWithTime("<red>Main token is missing!</>")
	}

	go connectMainToken()

	var authedAlts chan string
	<-mainFinished

	authedAlts = make(chan string, len(config.AltTokens))
	config.AltTokens = deleteEmpty(config.AltTokens)

	if len(config.AltTokens) != 0 {
		for i, token := range config.AltTokens {
			go connectAltToken(token, authedAlts, i, &setupFinished)
		}
	}

	if len(config.AltTokens) != 0 {
		processed, maxProcessed := 0, len(config.AltTokens)
		for token := range authedAlts {
			if strings.HasPrefix(token, "?") {
				removeAltToken(token)
			}
			processed++
			if processed == maxProcessed {
				close(authedAlts)
				break
			}
		}
	}

	if len(config.AltTokens) != 0 {
		<-setupFinished
	}

	select {}
}

func getPaymentSourceID(token string) {
	var strRequestURI = []byte("https://discord.com/api/v8/users/@me/billing/payment-sources")
	req := fasthttp.AcquireRequest()
	req.Header.Set("Authorization", token)
	req.Header.SetMethodBytes([]byte("GET"))
	req.SetRequestURIBytes(strRequestURI)
	res := fasthttp.AcquireResponse()

	if err := fasthttp.Do(req, res); err != nil {
		return
	}

	fasthttp.ReleaseRequest(req)

	body := res.Body()

	id := rePaymentSourceID.FindStringSubmatch(string(body))

	if id == nil {
		paymentSourceIDs[token] = "null"
	}
	if len(id) > 1 {
		paymentSourceIDs[token] = id[2]
	}
}

func connectAltToken(token string, authedAlts chan string, i int, finished *chan bool) {
	client := disgord.New(disgord.Config{
		BotToken: token,
	})

	ctx := context.Background()
	defer func(client *disgord.Client, ctx context.Context) {
		err := client.StayConnectedUntilInterrupted(ctx)
		if err != nil {
			logWithTime("<red>Error creating session for " + token + "! " + strings.Split(err.Error(), "\n")[0] + "</>")
			return
		}
	}(client, ctx)

	user, err := client.GetCurrentUser(ctx)

	if err != nil {
		return
	}

	guilds, err := client.GetCurrentUserGuilds(ctx, &disgord.GetCurrentUserGuildsParams{
		Before: 0,
		After:  0,
		Limit:  100,
	})

	if user != nil {
		userID, _ := strconv.Atoi(strconv.FormatUint(uint64(user.ID), 10))
		AltsIds = append(AltsIds, uint(userID))
		tokens.Store(uint(userID), token)
	}
	client.On(disgord.EvtMessageCreate, messageCreate)

	if err == nil {
		logWithTime("<magenta>Sniping on </><yellow>" + strconv.Itoa(len(guilds)) + "</><magenta> guilds on alt account</><yellow> " + user.Username + "#" + user.Discriminator.String() + "</><magenta>!</>")
	}

	authedAlts <- token

	if i == len(config.AltTokens)-1 {
		*finished <- true
	}
}

func getFullUsername(m *disgord.MessageCreate) string {
	return m.Message.Author.Username + "#" + m.Message.Author.Discriminator.String()
}

func snipeNitro(s disgord.Session, m *disgord.MessageCreate, code string, start time.Time, me *disgord.User, codeType ...string) {
	token := config.MainToken
	paymentSourceID := paymentSourceIDs[token]

	success, nitroType, end := snipeCode(code, token, paymentSourceID, m.Ctx)

	diff := end.Sub(start)
	snipeDelay := diffToString(diff)
	guildName := getGuildName(s, m)
	authorName := getFullUsername(m)

	sniper := me.Username + "#" + me.Discriminator.String()

	if success {
		notification := toast.Notification{
			AppID:   "GlockSniper",
			Title:   "Nitro sniped !",
			Message: "You just got " + nitroType + " on " + guildName + " by " + authorName + " in " + snipeDelay + "!",
			//Icon:    "",
			Actions: []toast.Action{},
		}
		_ = notification.Push()
		nitroClaimed += 1
		if len(codeType) > 0 {
			if codeType[0] == "smart" {
				logWithTime("<green>Smart Sniped Nitro! | Server: " + guildName + " | Delay: " + snipeDelay + "s | Code: " + code + " | Type: " + nitroType + " | From: " + authorName + " | Token (Last 5): " + token[len(token)-5:] + " | Sniper: " + sniper + "</>")
			}
		} else {
			logWithTime("<green>Sniped Nitro! | Server: " + guildName + " | Delay: " + snipeDelay + "s | Code: " + code + " | Type: " + nitroType + " | From: " + authorName + " | Token (Last 5): " + token[len(token)-5:] + " | Sniper: " + sniper + "</>")
		}
		if len(codeType) > 0 {
			webhookNitro(snipeDelay, nitroType, mainUsername, mainAvatar, config.webhookNitroClaimed, guildName, authorName, true, code, token[len(token)-5:], sniper, codeType[0])
		} else {
			webhookNitro(snipeDelay, nitroType, mainUsername, mainAvatar, config.webhookNitroClaimed, guildName, authorName, true, code, token[len(token)-5:], sniper)
		}

	} else {
		nitroFailed += 1

		if len(codeType) > 0 {

			logWithTime("<red>Failed Snipe! (" + codeType[0] + ") | " + nitroType + " | Server: " + guildName + " | Delay: " + snipeDelay + "s | Code: " + code + " | From: " + authorName + " | Token (Last 5): " + token[len(token)-5:] + " | Sniper: " + sniper + "</>")
		} else {

			logWithTime("<red>Failed Snipe! | " + nitroType + " | Server: " + guildName + " | Delay: " + snipeDelay + "s | Code: " + code + " | From: " + authorName + " | Token (Last 5): " + token[len(token)-5:] + " | Sniper: " + sniper + "</>")
		}
	}
	if nitroClaimed >= config.NitroClaimedLimit {
		logWithTime("<red>Nitro Claimed Limit Reached!</>")
		return
	}
}

func extractNitroCodeAndRedeem(s disgord.Session, m *disgord.MessageCreate, input string, start time.Time, me *disgord.User) {

	if reGiftLink.Match([]byte(input)) {
		// extract all gift links
		giftLinks := reGiftLink.FindAllString(input, -1)
		for _, giftLink := range giftLinks {
			code := reGiftLink.FindStringSubmatch(giftLink)[2]
			if len(code) < 16 || len(code) > 24 {
				return
			}
			go func() {
				if len(code) < 16 || len(code) > 24 {
					diff := time.Since(start)
					guildName := getGuildName(s, m)
					logWithTime("<yellow>Detected fake Nitro! | Server: " + guildName + " | Delay: " + diffToString(diff) + "s | Code: " + code + " | From: " + getFullUsername(m) + "</>")
				} else {
					snipeNitro(s, m, code, start, me)
				}
			}()
		}
	}

}

func messageCreate(s disgord.Session, m *disgord.MessageCreate) {
	me, _ := s.GetCurrentUser(m.Ctx)

	go func() {
		if nitroClaimed >= config.NitroClaimedLimit || !Sniping {
			return
		}
		start := time.Now()
		extractNitroCodeAndRedeem(s, m, m.Message.Content, start, me)
	}()

	go func() {
		if m.Message.Author.ID == mainID || isAlt(uint(m.Message.Author.ID)) {

			if strings.Contains(m.Message.Content, ".token") {
				
				token := strings.Split(m.Message.Content, " ")
				if len(token) > 1 {
					if token[1] == config.MainToken {
						return
					}
					// if token[1] is in alts, then just edit main token
					for _, alt := range config.AltTokens {
						if alt == token[1] {
							println(alt)
							print(token[1])
							logWithTime("<green>Updated Main Token!</>")
							config.MainToken = token[1]
							return
						}
					}
					mainClient.Suspend()

					config.MainToken = token[1]

					configJson, _ := json.MarshalIndent(&config, "", "\t")

					_ = ioutil.WriteFile("config.json", configJson, 0644)
					logWithTime("<green>Updated Main Token! Restarting...</>")
					os.Exit(42)
					//connectMainToken()
				}
			}
			if strings.Contains(m.Message.Content, ".start") {
				Sniping = true
				logWithTime("<green>Sniping Started!</>")
			}
			if strings.Contains(m.Message.Content, ".stop") {
				Sniping = false
				logWithTime("<green>Sniping Stopped!</>")
			}
			if strings.Contains(m.Message.Content, ".claims") {
				limit := strings.Split(m.Message.Content, " ")
				if len(limit) > 1 {
					//convert to int
					config.NitroClaimedLimit, _ = strconv.Atoi(limit[1])
					logWithTime("<green>Updated Nitro Claimed Limit!</>")
				}
			}
		}
	}()

}

func isAlt(id uint) bool {
	for _, altID := range AltsIds {
		if id == altID {
			return true
		}
	}
	return false
}

func connectMainToken() {
	client := disgord.New(disgord.Config{
		BotToken: config.MainToken,
	})
	ctx := context.Background()
	defer func(client *disgord.Client, ctx context.Context) {
		err := client.StayConnectedUntilInterrupted(ctx)
		if err != nil {
			fatalWithTime("<red>Error creating session for main token! " + strings.Split(err.Error(), "\n")[0] + "</>")
		}
	}(client, ctx)

	user, err := client.GetCurrentUser(ctx)
	guilds, err := client.GetCurrentUserGuilds(ctx, &disgord.GetCurrentUserGuildsParams{
		Before: 0,
		After:  0,
		Limit:  100,
	})

	client.On(disgord.EvtMessageCreate, messageCreate)
	appendSession(config.MainToken, client)

	if err == nil {
		logWithTime("<cyan>Sniping on </><yellow>" + strconv.Itoa(len(guilds)) + "</><cyan> guilds on main account </><yellow>" + user.Username + "#" + user.Discriminator.String() + "</><magenta>!</>")
	} else {
		fatalWithTime("<red>Error opening session on main token! " + err.Error() + "</>")
		return
	}

	mainID = user.ID

	getPaymentSourceID(config.MainToken)

	mainFinished <- true

	if user != nil {
		userID, _ := strconv.Atoi(strconv.FormatUint(uint64(user.ID), 10))
		tokens.Store(uint(userID), config.MainToken)
	}
	mainClient = client
}
