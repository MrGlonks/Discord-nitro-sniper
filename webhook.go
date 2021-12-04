package main

import (
	"time"

	"github.com/valyala/fasthttp"
)

func webhookNitro(delay, nitroType, username, authorAvatar, link, guildName, authorName string, success bool, code, tokenFive string, sniper string, codeType ...string) {
	title := "Failed Snipe!"
	if success {
		title = "Sniped Nitro!"
	}
	if len(codeType) > 0 {
		title += " (" + codeType[0] + ")"
	}
	const color string = "16749250"

	content := ""
	mention := ""

	body := `
	{
		"content": "` + content + `",
        "allowed_mentions": ` + mention + `,
		"embeds": [
			{
				"title": "` + title + `",
				"color": ` + color + `,
				"timestamp": "` + time.Now().Format(time.RFC3339) + `",
				"fields": [
					{
						"name": "Type",
						"value": "` + "`" + nitroType + "`" + `",
						"inline": true
					},
					{
						"name": "Delay",
						"value": "` + "`" + delay + "s`" + `",
						"inline": true
					},
					{
						"name": "Guild",
						"value": "` + "`" + guildName + "`" + `",
						"inline": false
					},
					{
						"name": "Author",
						"value": "` + "`" + authorName + "`" + `",
						"inline": true
					},
					{
						"name": "Code",
						"value": "` + "`" + code + "`" + `",
						"inline": false
					},
					{
						"name": "Token Ending",
						"value": "` + "`" + tokenFive + "`" + `",
						"inline": true
					},
					{
						"name": "Sniper",
						"value": "` + "`" + sniper + "`" + `",
						"inline": true
					}
				],
		  		"author": {
					"name": "` + username + `",
					"icon_url": "` + authorAvatar + `"
				},
		  		"footer": {
					"text": "GLOCK SNIPER"
		  		}
			}
	 	],
		"username":  "GLOCK SNIPER"
	}
	`

	req := fasthttp.AcquireRequest()
	req.Header.SetContentType("application/json")
	req.SetBodyString(body)
	req.Header.SetMethod("POST")
	req.SetRequestURI(link)
	res := fasthttp.AcquireResponse()
	if err := fasthttp.Do(req, res); err != nil {
		return
	}
	fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)
}
