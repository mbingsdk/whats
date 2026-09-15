package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"unicode/utf8"
)

type SendOutcome struct{ State, ProviderID, Code string }

// SendText is not an arbitrary POST proxy. It sends at most one request and
// never shares the read client's retry classification with an ambiguous send.
func (g *Graph) SendText(ctx context.Context, phone, recipient, body, correlation string) SendOutcome {
	if !numericID.MatchString(phone) || !numericID.MatchString(recipient) || body == "" || utf8.RuneCountInString(body) > 4096 || g.version != "v26.0" {
		return SendOutcome{State: "REJECTED", Code: "INVALID_TEXT_REQUEST"}
	}
	payload, _ := json.Marshal(map[string]any{"messaging_product": "whatsapp", "recipient_type": "individual", "to": recipient, "type": "text", "text": map[string]any{"body": body, "preview_url": false}, "biz_opaque_callback_data": correlation})
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, g.base+"/"+g.version+"/"+phone+"/messages", bytes.NewReader(payload))
	if e != nil {
		return SendOutcome{State: "REJECTED", Code: "INVALID_TEXT_REQUEST"}
	}
	req.Header.Set("Authorization", "Bearer "+string(g.token))
	req.Header.Set("Content-Type", "application/json")
	response, e := g.client.Do(req)
	if e != nil {
		return SendOutcome{State: "UNCERTAIN", Code: "DISPATCH_UNCERTAIN"}
	}
	defer response.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(response.Body, 65537))
	if e != nil || len(raw) > 65536 {
		return SendOutcome{State: "UNCERTAIN", Code: "DISPATCH_UNCERTAIN"}
	}
	var result struct {
		Messages []struct{ ID string }
		Error    *struct{ Code int }
	}
	if json.Unmarshal(raw, &result) != nil {
		return SendOutcome{State: "UNCERTAIN", Code: "DISPATCH_UNCERTAIN"}
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 && result.Error == nil && len(result.Messages) == 1 && len(result.Messages[0].ID) > 0 && len(result.Messages[0].ID) <= 512 {
		return SendOutcome{State: "ACCEPTED", ProviderID: result.Messages[0].ID}
	}
	if response.StatusCode >= 400 && response.StatusCode < 500 && result.Error != nil {
		switch result.Error.Code {
		case 100, 190, 10, 200, 131047:
			return SendOutcome{State: "REJECTED", Code: "META_DEFINITE_REJECTION"}
		case 130429, 131056:
			return SendOutcome{State: "REJECTED", Code: "META_RATE_LIMITED"}
		}
	}
	return SendOutcome{State: "UNCERTAIN", Code: "DISPATCH_UNCERTAIN"}
}
