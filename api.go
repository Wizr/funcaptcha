package funcaptcha

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
)

type GetTokenOptions struct {
	PKey     string            `json:"pkey"`
	SURL     string            `json:"surl,omitempty"`
	Data     map[string]string `json:"data,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Site     string            `json:"site,omitempty"`
	Location string            `json:"location,omitempty"`
	Proxy    string            `json:"proxy,omitempty"`
}

type GetTokenResult struct {
	ChallengeURL          string `json:"challenge_url"`
	ChallengeURLCDN       string `json:"challenge_url_cdn"`
	ChallengeURLCDNSRI    string `json:"challenge_url_cdn_sri"`
	DisableDefaultStyling bool   `json:"disable_default_styling"`
	IFrameHeight          int    `json:"iframe_height"`
	IFrameWidth           int    `json:"iframe_width"`
	KBio                  bool   `json:"kbio"`
	MBio                  bool   `json:"mbio"`
	NoScript              string `json:"noscript"`
	TBio                  bool   `json:"tbio"`
	Token                 string `json:"token"`
}

var (
	jar     = tls_client.NewCookieJar()
	options = []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(360),
		tls_client.WithClientProfile(tls_client.Safari_IOS_16_0),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar), // create cookieJar instance and pass it as argument
	}
	client, _ = tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
)

func GetToken(options *GetTokenOptions) (GetTokenResult, error) {
	if options.SURL == "" {
		options.SURL = "https://client-api.arkoselabs.com"
	}
	if options.Headers == nil {
		options.Headers = make(map[string]string)
	}
	if _, ok := options.Headers["User-Agent"]; !ok {
		options.Headers["User-Agent"] = DEFAULT_USER_AGENT
	}

	options.Headers["Accept-Language"] = "en-US,en;q=0.9"
	options.Headers["Sec-Fetch-Site"] = "same-origin"
	options.Headers["Accept"] = "*/*"
	options.Headers["Content-Type"] = "application/x-www-form-urlencoded; charset=UTF-8"
	options.Headers["sec-fetch-mode"] = "cors"

	if options.Site != "" {
		options.Headers["Origin"] = options.SURL
		options.Headers["Referer"] = fmt.Sprintf("%s/v2/%s/1.4.3/enforcement.%s.html", options.SURL, options.PKey, Random())
	}

	ua := options.Headers["User-Agent"]
	formData := url.Values{
		"bda":         {GetBda(ua, options.Headers["Referer"], options.Location)},
		"public_key":  {options.PKey},
		"site":        {options.Site},
		"userbrowser": {ua},
		"rnd":         {strconv.FormatFloat(rand.Float64(), 'f', -1, 64)},
	}

	if options.Site == "" {
		formData.Del("site")
	}

	for key, value := range options.Data {
		formData.Add("data["+key+"]", value)
	}

	form := strings.ReplaceAll(formData.Encode(), "+", "%20")
	form = strings.ReplaceAll(form, "%28", "(")
	form = strings.ReplaceAll(form, "%29", ")")
	req, err := http.NewRequest("POST", options.SURL+"/fc/gt2/public_key/"+options.PKey, bytes.NewBufferString(form))
	if err != nil {
		return GetTokenResult{}, err
	}

	for key, value := range options.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return GetTokenResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GetTokenResult{}, err
	}

	var result GetTokenResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return GetTokenResult{}, err
	}

	return result, nil
}

type OpenAiRequest struct {
	Request *http.Request
	Client  *tls_client.HttpClient
}

func (this *OpenAiRequest) GetToken() (string, error) {
	resp, err := (*this.Client).Do(this.Request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result GetTokenResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	return result.Token, nil
}

func NewOpenAiRequestV1() (*OpenAiRequest, error) {
	surl := "https://tcr9i.chat.openai.com"
	pkey := "35536E1E-65B4-4D96-9D97-6ADB7EFF8147"

	formData := url.Values{
		"bda": {GetBda(DEFAULT_USER_AGENT,
			fmt.Sprintf("%s/v2/%s/1.4.3/enforcement.%s.html",
				surl, pkey, Random()), "")},
		"public_key":   {"35536E1E-65B4-4D96-9D97-6ADB7EFF8147"},
		"site":         {"https://chat.openai.com"},
		"userbrowser":  {DEFAULT_USER_AGENT},
		"capi_version": {"1.5.2"},
		"capi_mode":    {"lightbox"},
		"style_theme":  {"default"},
		"rnd":          {strconv.FormatFloat(rand.Float64(), 'f', -1, 64)},
	}

	form := strings.ReplaceAll(formData.Encode(), "+", "%20")
	form = strings.ReplaceAll(form, "%28", "(")
	form = strings.ReplaceAll(form, "%29", ")")
	req, err := http.NewRequest("POST", surl+"/fc/gt2/public_key/"+pkey, bytes.NewBufferString(form))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Origin", surl)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("sec-fetch-mode", "cors")
	return &OpenAiRequest{
		Request: req,
		Client:  &client,
	}, nil
}

func GetOpenAITokenV1() (string, error) {
	req, err := NewOpenAiRequestV1()
	if err != nil {
		return "", err
	}
	return req.GetToken()
}

func NewOpenAiRequestV2() (*OpenAiRequest, error) {
	// generate timestamp in 1687790752 format
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano()/1000000000)
	bx := fmt.Sprintf(`[{"key":"api_type","value":"js"},{"key":"p","value":1},{"key":"f","value":"9711bd3695defe0844fb8fd8a722f38b"},{"key":"n","value":"%s"},{"key":"wh","value":"80b13fd48b8da8e4157eeb6f9e9fbedb|5ab5738955e0611421b686bc95655ad0"},{"key":"enhanced_fp","value":[{"key":"webgl_extensions","value":null},{"key":"webgl_extensions_hash","value":null},{"key":"webgl_renderer","value":null},{"key":"webgl_vendor","value":null},{"key":"webgl_version","value":null},{"key":"webgl_shading_language_version","value":null},{"key":"webgl_aliased_line_width_range","value":null},{"key":"webgl_aliased_point_size_range","value":null},{"key":"webgl_antialiasing","value":null},{"key":"webgl_bits","value":null},{"key":"webgl_max_params","value":null},{"key":"webgl_max_viewport_dims","value":null},{"key":"webgl_unmasked_vendor","value":null},{"key":"webgl_unmasked_renderer","value":null},{"key":"webgl_vsf_params","value":null},{"key":"webgl_vsi_params","value":null},{"key":"webgl_fsf_params","value":null},{"key":"webgl_fsi_params","value":null},{"key":"webgl_hash_webgl","value":null},{"key":"user_agent_data_brands","value":null},{"key":"user_agent_data_mobile","value":null},{"key":"navigator_connection_downlink","value":null},{"key":"navigator_connection_downlink_max","value":null},{"key":"network_info_rtt","value":null},{"key":"network_info_save_data","value":null},{"key":"network_info_rtt_type","value":null},{"key":"screen_pixel_depth","value":24},{"key":"navigator_device_memory","value":null},{"key":"navigator_languages","value":"en-US,en"},{"key":"window_inner_width","value":0},{"key":"window_inner_height","value":0},{"key":"window_outer_width","value":0},{"key":"window_outer_height","value":0},{"key":"browser_detection_firefox","value":true},{"key":"browser_detection_brave","value":false},{"key":"audio_codecs","value":"{\"ogg\":\"probably\",\"mp3\":\"maybe\",\"wav\":\"probably\",\"m4a\":\"maybe\",\"aac\":\"maybe\"}"},{"key":"video_codecs","value":"{\"ogg\":\"probably\",\"h264\":\"probably\",\"webm\":\"probably\",\"mpeg4v\":\"\",\"mpeg4a\":\"\",\"theora\":\"\"}"},{"key":"media_query_dark_mode","value":false},{"key":"headless_browser_phantom","value":false},{"key":"headless_browser_selenium","value":false},{"key":"headless_browser_nightmare_js","value":false},{"key":"document__referrer","value":""},{"key":"window__ancestor_origins","value":null},{"key":"window__tree_index","value":[1]},{"key":"window__tree_structure","value":"[[],[]]"},{"key":"window__location_href","value":"https://tcr9i.chat.openai.com/v2/1.5.2/enforcement.64b3a4e29686f93d52816249ecbf9857.html#35536E1E-65B4-4D96-9D97-6ADB7EFF8147"},{"key":"client_config__sitedata_location_href","value":"https://chat.openai.com/"},{"key":"client_config__surl","value":"https://tcr9i.chat.openai.com"},{"key":"mobile_sdk__is_sdk"},{"key":"client_config__language","value":null},{"key":"audio_fingerprint","value":"35.73833402246237"}]},{"key":"fe","value":["DNT:1","L:en-US","D:24","PR:1","S:0,0","AS:false","TO:0","SS:true","LS:true","IDB:true","B:false","ODB:false","CPUC:unknown","PK:Linux x86_64","CFP:330110783","FR:false","FOS:false","FB:false","JSF:Arial,Arial Narrow,Bitstream Vera Sans Mono,Bookman Old Style,Century Schoolbook,Courier,Courier New,Helvetica,MS Gothic,MS PGothic,Palatino,Palatino Linotype,Times,Times New Roman","P:Chrome PDF Viewer,Chromium PDF Viewer,Microsoft Edge PDF Viewer,PDF Viewer,WebKit built-in PDF","T:0,false,false","H:2","SWF:false"]},{"key":"ife_hash","value":"2a007a5daef41ee943d5fc73a0a8c312"},{"key":"cs","value":1},{"key":"jsbd","value":"{\"HL\":2,\"NCE\":true,\"DT\":\"\",\"NWD\":\"false\",\"DOTO\":1,\"DMTO\":1}"}]`,
		base64.StdEncoding.EncodeToString([]byte(timestamp)))
	// var bt = new Date() ['getTime']() / 1000
	bt := time.Now().UnixMicro() / 1000000
	// bw = Math.round(bt - (bt % 21600)
	bw := strconv.FormatInt(bt-(bt%21600), 10)
	bv := "Mozilla/5.0 (X11; Linux x86_64; rv:114.0) Gecko/20100101 Firefox/114.0"
	bda := Encrypt(bx, bv+bw)
	bda = base64.StdEncoding.EncodeToString([]byte(bda))
	form := url.Values{
		"bda":          {bda},
		"public_key":   {"35536E1E-65B4-4D96-9D97-6ADB7EFF8147"},
		"site":         {"https://chat.openai.com"},
		"userbrowser":  {bv},
		"capi_version": {"1.5.2"},
		"capi_mode":    {"lightbox"},
		"style_theme":  {"default"},
		"rnd":          {strconv.FormatFloat(rand.Float64(), 'f', -1, 64)},
	}
	req, _ := http.NewRequest(http.MethodPost, "https://tcr9i.chat.openai.com/fc/gt2/public_key/35536E1E-65B4-4D96-9D97-6ADB7EFF8147", strings.NewReader(form.Encode()))
	req.Header.Set("Host", "tcr9i.chat.openai.com")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; rv:114.0) Gecko/20100101 Firefox/114.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Origin", "https://tcr9i.chat.openai.com")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://tcr9i.chat.openai.com/v2/1.5.2/enforcement.64b3a4e29686f93d52816249ecbf9857.html")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("TE", "trailers")
	return &OpenAiRequest{
		Request: req,
		Client:  &client,
	}, nil
}

func GetOpenAITokenV2() (string, error) {
	req, err := NewOpenAiRequestV2()
	if err != nil {
		return "", err
	}
	return req.GetToken()
}
