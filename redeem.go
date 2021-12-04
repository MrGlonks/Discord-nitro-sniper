package main

import (
	"context"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

func snipeCode(nitroCode, token, paymentSourceID string, ctx context.Context) (bool, string, time.Time) {

	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()

	req.Header.SetContentType("application/json")
	req.Header.Set("Connection", "keep-alive")
	req.Header.SetMethod("POST")

	println(paymentSourceIDs)
	req.Header.Set("authorization", config.MainToken)
	req.SetBodyString(`{"channel_id": null ,"payment_source_id": ` + paymentSourceIDs[config.MainToken] + `}`)

	var strRequest = "https://discord.com/api/v9/entitlements/gift-codes/" + nitroCode + "/redeem"

	req.SetRequestURI(strRequest)

	if err := fasthttp.Do(req, res); err != nil {
		return false, err.Error(), time.Now()
	}

	end := time.Now()

	body := res.Body()

	bodyString := string(body)

	if strings.Contains(strings.ToLower(bodyString), "nitro") {
		nitroType := strings.Split(strings.Split(bodyString, `name": "`)[1], `"`)[0]
		return true, nitroType, end
	}
	if strings.Contains(strings.ToLower(bodyString), "limited") {
		return false, "You are being rate limited", end
	}
	response := bodyString
	if len(strings.Split(bodyString, `{"message": "`)) > 0 {
		response = strings.Split(strings.Split(bodyString, `{"message": "`)[1], `"`)[0]
	}
	response = strings.ReplaceAll(response, ".", "")
	return false, response, end
}
