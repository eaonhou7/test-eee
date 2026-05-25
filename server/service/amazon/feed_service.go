package amazon

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	amazonModel "github.com/flipped-aurora/gin-vue-admin/server/model/amazon"
)

type AmazonFeedService struct{}

type FeedDocumentDetail struct {
	DocumentID           string                 `json:"documentId"`
	URL                  string                 `json:"url"`
	CompressionAlgorithm string                 `json:"compressionAlgorithm"`
	Response             map[string]interface{} `json:"response"`
}

type FeedStatusDetail struct {
	FeedID           string                 `json:"feedId"`
	ProcessingStatus string                 `json:"processingStatus"`
	ResultDocumentID string                 `json:"resultDocumentId"`
	Response         map[string]interface{} `json:"response"`
}

func (s *AmazonFeedService) CreateFeedDocument(ctx context.Context, store amazonModel.StoreAccount, contentType string) (FeedDocumentDetail, error) {
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/json; charset=UTF-8"
	}
	resp, _, err := newSPAPIClient().requestJSON(ctx, store, http.MethodPost, "/feeds/2021-06-30/documents", nil, map[string]interface{}{
		"contentType": contentType,
	}, nil)
	if err != nil {
		return FeedDocumentDetail{}, err
	}
	payload := extractPayloadMap(resp)
	return FeedDocumentDetail{
		DocumentID: strings.TrimSpace(fmt.Sprintf("%v", payload["feedDocumentId"])),
		URL:        strings.TrimSpace(fmt.Sprintf("%v", payload["url"])),
		Response:   resp,
	}, nil
}

func (s *AmazonFeedService) UploadDocument(ctx context.Context, uploadURL string, payload []byte, contentType string) error {
	if strings.TrimSpace(uploadURL) == "" {
		return errors.New("Amazon 未返回 feed 文档上传地址")
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/json; charset=UTF-8"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", contentType)
	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("上传 feed 文档失败 (%d): %s", resp.StatusCode, string(raw))
	}
	return nil
}

func (s *AmazonFeedService) CreateFeed(ctx context.Context, store amazonModel.StoreAccount, feedType, documentID string, marketplaceIDs []string) (string, map[string]interface{}, error) {
	resp, _, err := newSPAPIClient().requestJSON(ctx, store, http.MethodPost, "/feeds/2021-06-30/feeds", nil, map[string]interface{}{
		"feedType":            strings.TrimSpace(feedType),
		"marketplaceIds":      marketplaceIDs,
		"inputFeedDocumentId": strings.TrimSpace(documentID),
	}, nil)
	if err != nil {
		return "", nil, err
	}
	payload := extractPayloadMap(resp)
	return strings.TrimSpace(fmt.Sprintf("%v", payload["feedId"])), resp, nil
}

func (s *AmazonFeedService) RefreshFeedStatus(ctx context.Context, store amazonModel.StoreAccount, feedID string) (FeedStatusDetail, []byte, error) {
	resp, raw, err := newSPAPIClient().requestJSON(ctx, store, http.MethodGet, "/feeds/2021-06-30/feeds/"+url.PathEscape(strings.TrimSpace(feedID)), nil, nil, nil)
	if err != nil {
		return FeedStatusDetail{}, nil, err
	}
	payload := extractPayloadMap(resp)
	processingStatus := strings.TrimSpace(fmt.Sprintf("%v", payload["processingStatus"]))
	if processingStatus == "" {
		processingStatus = strings.TrimSpace(fmt.Sprintf("%v", payload["processing_status"]))
	}
	return FeedStatusDetail{
		FeedID:           strings.TrimSpace(feedID),
		ProcessingStatus: processingStatus,
		ResultDocumentID: strings.TrimSpace(fmt.Sprintf("%v", payload["resultFeedDocumentId"])),
		Response:         resp,
	}, raw, nil
}

func (s *AmazonFeedService) GetFeedDocument(ctx context.Context, store amazonModel.StoreAccount, documentID string) (FeedDocumentDetail, error) {
	resp, _, err := newSPAPIClient().requestJSON(ctx, store, http.MethodGet, "/feeds/2021-06-30/documents/"+url.PathEscape(strings.TrimSpace(documentID)), nil, nil, nil)
	if err != nil {
		return FeedDocumentDetail{}, err
	}
	payload := extractPayloadMap(resp)
	return FeedDocumentDetail{
		DocumentID:           strings.TrimSpace(documentID),
		URL:                  strings.TrimSpace(fmt.Sprintf("%v", payload["url"])),
		CompressionAlgorithm: strings.TrimSpace(fmt.Sprintf("%v", payload["compressionAlgorithm"])),
		Response:             resp,
	}, nil
}

func (s *AmazonFeedService) DownloadDocument(ctx context.Context, document FeedDocumentDetail) ([]byte, error) {
	if strings.TrimSpace(document.URL) == "" {
		return nil, errors.New("Amazon 未返回结果文档下载地址")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, document.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Timeout: 90 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("下载 Amazon 结果文档失败 (%d): %s", resp.StatusCode, string(raw))
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(document.CompressionAlgorithm), "GZIP") {
		return raw, nil
	}
	reader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
