package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// minecraftAuthURL is the URL that an authentication request is made to to get an encoded JWT claim chain.
const minecraftAuthURL = `https://multiplayer.minecraft.net/authentication`

// RequestMinecraftChain requests a fully processed Minecraft JWT chain using the XSTS token passed, and the
// ECDSA private key of the client. This key will later be used to initialise encryption, and must be saved
// for when packets need to be decrypted/encrypted.
func RequestMinecraftChain(ctx context.Context, token *XBLToken, key *ecdsa.PrivateKey, c *http.Client) (string, error) {
	if c == nil {
		c = &http.Client{}
	}

	data, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)

	// The body of the requests holds a JSON object with one key in it, the 'identityPublicKey', which holds
	// the public key data of the private key passed.
	body := `{"identityPublicKey":"` + base64.StdEncoding.EncodeToString(data) + `"}`
	request, _ := http.NewRequestWithContext(ctx, "POST", minecraftAuthURL, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	// The Authorization header is important in particular. It is composed of the 'uhs' found in the XSTS
	// token, and the Token it holds itself.
	token.SetAuthHeader(request)
	request.Header.Set("User-Agent", "MCPE/Android")
	request.Header.Set("Client-Version", protocol.CurrentVersion)

	resp, err := c.Do(request)
	if err != nil {
		return "", fmt.Errorf("POST %v: %w", minecraftAuthURL, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("POST %v: %v", minecraftAuthURL, resp.Status)
	}
	data, err = io.ReadAll(resp.Body)
	c.CloseIdleConnections()
	return string(data), err
}
