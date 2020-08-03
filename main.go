package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fujiwara/ridge"
	"github.com/pkg/errors"

	mackerel "github.com/mackerelio/mackerel-client-go"
)

const (
	commonAttrHeaderName = "X-Amz-Firehose-Common-Attributes"
	requestIDHeaderName  = "X-Amz-Firehose-Request-Id"
	accessKeyHeaderName  = "X-Amz-Firehose-Access-Key"
)

// FirehoseCommonAttributes represents common attributes (metadata).
// https://docs.aws.amazon.com/ja_jp/firehose/latest/dev/httpdeliveryrequestresponse.html#requestformat
type FirehoseCommonAttributes struct {
	CommonAttributes map[string]string `json:"commonAttributes"`
}

// RequestBody represents request body.
type RequestBody struct {
	RequestID string   `json:"requestId,omitempty"`
	Timestamp int64    `json:"timestamp,omitempty"`
	Records   []Record `json:"records,omitempty"`
}

// Record represents records in request body.
type Record struct {
	Data []byte `json:"data"`
}

// ResponseBody represents response body.
// https://docs.aws.amazon.com/ja_jp/firehose/latest/dev/httpdeliveryrequestresponse.html#responseformat
type ResponseBody struct {
	RequestID    string `json:"requestId,omitempty"`
	Timestamp    int64  `json:"timestamp,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

func main() {
	var mux = http.NewServeMux()
	mux.HandleFunc("/service", handleServiceMetrics)
	mux.HandleFunc("/", handleRoot)
	ridge.Run(":8080", "/", mux)
}

func parseRequest(r *http.Request) (string, string, *RequestBody, error) {
	var attrs FirehoseCommonAttributes
	if err := json.Unmarshal([]byte(r.Header.Get(commonAttrHeaderName)), &attrs); err != nil {
		return "", "", nil, fmt.Errorf("[error] failed to parse request header %s: %s", commonAttrHeaderName, err)
	}
	apiKey := r.Header.Get(accessKeyHeaderName)
	if apiKey == "" {
		return "", "", nil, errors.New("[error] apikey not found in access key")
	}
	service := attrs.CommonAttributes["service"]
	if service == "" {
		return "", "", nil, errors.New("[error] service not found in attributes")
	}
	var body RequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return "", "", nil, fmt.Errorf("[error] failed to decode request body: %s", err)
	}
	return apiKey, service, &body, nil
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("content-type", "text/plain")
	w.Write([]byte("OK\n"))
}

func handleServiceMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("content-type", "application/json")
	respBody := ResponseBody{
		RequestID: r.Header.Get(requestIDHeaderName),
	}
	defer func() {
		respBody.Timestamp = time.Now().UnixNano() / int64(time.Millisecond)
		if e := respBody.ErrorMessage; e != "" {
			log.Printf("[error] error:%s", e)
		}
		json.NewEncoder(w).Encode(respBody)
	}()
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		respBody.ErrorMessage = "POST method required"
		return
	}

	apiKey, service, reqBody, err := parseRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		respBody.ErrorMessage = err.Error()
		return
	}
	log.Printf(
		"[info] accept service metrics requestId:%s timestamp:%d records:%d",
		reqBody.RequestID,
		reqBody.Timestamp,
		len(reqBody.Records),
	)

	if err := postServiceMetrics(apiKey, service, reqBody.Records); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		respBody.ErrorMessage = err.Error()
		return
	}
}

func postServiceMetrics(apiKey string, service string, records []Record) error {
	var mvs []*mackerel.MetricValue
	for _, record := range records {
		var mv mackerel.MetricValue
		if err := json.Unmarshal(record.Data, &mv); err == nil {
			mvs = append(mvs, &mv)
		} else if err := parseMetricLine(record.Data, &mv); err == nil {
			mvs = append(mvs, &mv)
		} else {
			log.Printf("[warn] failed to parse record as metricValue: %s %s", err, record.Data)
		}
	}
	if len(mvs) == 0 {
		log.Println("[warn] no service metric values to post")
		return nil
	}

	c := mackerel.NewClient(apiKey)
	if err := c.PostServiceMetricValues(service, mvs); err != nil {
		return errors.Wrapf(err, "[error] failed to post metrics values to service %s", service)
	}
	b, _ := json.Marshal(mvs)
	log.Printf("[debug] post metricValue to service %s: %s", service, string(b))
	return nil
}

func parseMetricLine(b []byte, mv *mackerel.MetricValue) error {
	s := strings.TrimSpace(string(b))
	cols := strings.SplitN(s, "\t", 3)
	if len(cols) < 3 {
		return errors.New("invalid metric format. insufficient columns")
	}
	name, value, timestamp := cols[0], cols[1], cols[2]
	mv.Name = name

	if v, err := strconv.ParseFloat(value, 64); err != nil {
		return fmt.Errorf("invalid metric value: %s", value)
	} else {
		mv.Value = v
	}

	if ts, err := strconv.ParseInt(timestamp, 10, 64); err != nil {
		return fmt.Errorf("invalid metric time: %s", timestamp)
	} else {
		mv.Time = ts
	}
	return nil
}
