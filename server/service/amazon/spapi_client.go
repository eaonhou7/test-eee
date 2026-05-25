package amazon

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
)

type spapiRegionMeta struct {
	APIBase           string
	SellerCentralBase string
	AWSRegion         string
}

type spapiClient struct {
	httpClient *http.Client
}

var spapiRegions = map[string]spapiRegionMeta{
	"NA": {
		APIBase:           "https://sellingpartnerapi-na.amazon.com",
		SellerCentralBase: "https://sellercentral.amazon.com",
		AWSRegion:         "us-east-1",
	},
	"EU": {
		APIBase:           "https://sellingpartnerapi-eu.amazon.com",
		SellerCentralBase: "https://sellercentral.amazon.co.uk",
		AWSRegion:         "eu-west-1",
	},
	"FE": {
		APIBase:           "https://sellingpartnerapi-fe.amazon.com",
		SellerCentralBase: "https://sellercentral.amazon.co.jp",
		AWSRegion:         "us-west-2",
	},
}

func newSPAPIClient() *spapiClient {
	return &spapiClient{
		httpClient: &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *spapiClient) buildAuthorizationURL(store amazonModel.StoreAccount, state string) (string, error) {
	meta := regionMeta(store.Region)
	applicationID := strings.TrimSpace(global.GVA_CONFIG.Amazon.ApplicationID)
	redirectURI := strings.TrimSpace(global.GVA_CONFIG.Amazon.OAuthRedirectURI)
	if applicationID == "" {
		return "", errors.New("amazon.application-id 未配置")
	}
	if redirectURI == "" {
		return "", errors.New("amazon.oauth-redirect-uri 未配置")
	}
	params := url.Values{}
	params.Set("application_id", applicationID)
	params.Set("state", state)
	params.Set("version", "beta")
	params.Set("redirect_uri", redirectURI)
	return meta.SellerCentralBase + "/apps/authorize/consent?" + params.Encode(), nil
}

func (c *spapiClient) exchangeAuthCode(ctx context.Context, authCode string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", strings.TrimSpace(authCode))
	form.Set("client_id", strings.TrimSpace(global.GVA_CONFIG.Amazon.LWAClientID))
	form.Set("client_secret", strings.TrimSpace(global.GVA_CONFIG.Amazon.LWAClientSecret))
	redirectURI := strings.TrimSpace(global.GVA_CONFIG.Amazon.OAuthRedirectURI)
	if redirectURI != "" {
		form.Set("redirect_uri", redirectURI)
	}
	respBody, err := c.lwaTokenRequest(ctx, form)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(fmt.Sprintf("%v", respBody["refresh_token"])), nil
}

func (c *spapiClient) exchangeRefreshToken(ctx context.Context, refreshToken string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", strings.TrimSpace(refreshToken))
	form.Set("client_id", strings.TrimSpace(global.GVA_CONFIG.Amazon.LWAClientID))
	form.Set("client_secret", strings.TrimSpace(global.GVA_CONFIG.Amazon.LWAClientSecret))
	respBody, err := c.lwaTokenRequest(ctx, form)
	if err != nil {
		return "", err
	}
	accessToken := strings.TrimSpace(fmt.Sprintf("%v", respBody["access_token"]))
	if accessToken == "" {
		return "", errors.New("Amazon LWA 未返回 access_token")
	}
	return accessToken, nil
}

func (c *spapiClient) lwaTokenRequest(ctx context.Context, form url.Values) (map[string]interface{}, error) {
	if strings.TrimSpace(global.GVA_CONFIG.Amazon.LWAClientID) == "" || strings.TrimSpace(global.GVA_CONFIG.Amazon.LWAClientSecret) == "" {
		return nil, errors.New("amazon.lwa-client-id 或 amazon.lwa-client-secret 未配置")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.amazon.com/auth/o2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var parsed map[string]interface{}
	_ = json.Unmarshal(raw, &parsed)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Amazon LWA 请求失败 (%d): %s", resp.StatusCode, string(raw))
	}
	return parsed, nil
}

func (c *spapiClient) requestJSON(ctx context.Context, store amazonModel.StoreAccount, method, apiPath string, query url.Values, body interface{}, extraHeaders map[string]string) (map[string]interface{}, []byte, error) {
	refreshToken, err := decryptAmazonSecret(store.RefreshTokenEncrypted)
	if err != nil {
		return nil, nil, err
	}
	accessToken, err := c.exchangeRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, nil, err
	}
	meta := regionMeta(store.Region)
	rawBody := []byte(nil)
	if body != nil {
		rawBody, err = json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
	}
	fullURL := meta.APIBase + apiPath
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(rawBody))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("user-agent", "gva-amazon-tool/1.0")
	req.Header.Set("x-amz-access-token", accessToken)
	if len(rawBody) > 0 {
		req.Header.Set("content-type", "application/json")
	}
	for key, value := range extraHeaders {
		if strings.TrimSpace(value) != "" {
			req.Header.Set(key, value)
		}
	}
	creds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		strings.TrimSpace(global.GVA_CONFIG.Amazon.AWSAccessKeyID),
		strings.TrimSpace(global.GVA_CONFIG.Amazon.AWSSecretKey),
		"",
	))
	if strings.TrimSpace(global.GVA_CONFIG.Amazon.AWSAccessKeyID) == "" || strings.TrimSpace(global.GVA_CONFIG.Amazon.AWSSecretKey) == "" {
		return nil, nil, errors.New("amazon.aws-access-key-id 或 amazon.aws-secret-access-key 未配置")
	}
	signer := v4.NewSigner()
	credsValue, err := creds.Retrieve(ctx)
	if err != nil {
		return nil, nil, err
	}
	if err := signer.SignHTTP(ctx, credsValue, req, sha256Hex(rawBody), "execute-api", meta.AWSRegion, time.Now()); err != nil {
		return nil, nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	var parsed map[string]interface{}
	_ = json.Unmarshal(responseBody, &parsed)
	if resp.StatusCode >= 300 {
		return parsed, responseBody, fmt.Errorf("Amazon SP-API 请求失败 (%d): %s", resp.StatusCode, string(responseBody))
	}
	return parsed, responseBody, nil
}

func decryptAmazonSecret(cipherText string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	if len(decoded) < 12 {
		return "", errors.New("加密内容无效")
	}
	keys := amazonEncryptionKeyCandidates()
	if len(keys) == 0 {
		return "", errors.New("amazon.encryption-key 或 jwt.signing-key 未配置")
	}
	var lastErr error
	for _, key := range keys {
		block, err := aes.NewCipher(hashAmazonEncryptionKey(key))
		if err != nil {
			lastErr = err
			continue
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			lastErr = err
			continue
		}
		if len(decoded) < gcm.NonceSize() {
			lastErr = errors.New("加密内容无效")
			continue
		}
		nonce := decoded[:gcm.NonceSize()]
		plain, err := gcm.Open(nil, nonce, decoded[gcm.NonceSize():], nil)
		if err == nil {
			return string(plain), nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", errors.New("解密 Amazon 凭证失败")
}

func encryptAmazonSecret(plainText string) (string, error) {
	key, err := primaryAmazonEncryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(hashAmazonEncryptionKey(key))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plainText), nil)), nil
}

func hashAmazonEncryptionKey(key string) []byte {
	hash := sha256.Sum256([]byte(key))
	return hash[:]
}

func primaryAmazonEncryptionKey() (string, error) {
	keys := amazonEncryptionKeyCandidates()
	if len(keys) == 0 {
		return "", errors.New("amazon.encryption-key 或 jwt.signing-key 未配置")
	}
	return keys[0], nil
}

func amazonEncryptionKeyCandidates() []string {
	keys := make([]string, 0, 2)
	appendIfMissing := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		for _, item := range keys {
			if item == trimmed {
				return
			}
		}
		keys = append(keys, trimmed)
	}

	appendIfMissing(global.GVA_CONFIG.Amazon.EncryptionKey)
	appendIfMissing(global.GVA_CONFIG.JWT.SigningKey)
	return keys
}

func sha256Hex(value []byte) string {
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}

func regionMeta(region string) spapiRegionMeta {
	meta, ok := spapiRegions[strings.ToUpper(strings.TrimSpace(region))]
	if !ok {
		return spapiRegions["NA"]
	}
	return meta
}

func randomStateToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
