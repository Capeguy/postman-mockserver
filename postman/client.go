package postman

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	. "github.com/dvincenz/postman-mockserver/common"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Client struct {
	BaseURL    *url.URL
	UserAgent  string
	httpClient *http.Client
}

func getCollection(collectionId string) (postmanCollection, error) {
	responseJson, err := getFromPostman("/collections/" + collectionId)
	if err != nil {
		return postmanCollection{}, err
	}
	var postmanCollection = postmanCollection{}
	substr := "\"info\""
	pos := strings.Index(responseJson, substr)
	updatedJson := responseJson[pos:]
	updatedJson = updatedJson[:len(updatedJson)-1]
	updatedJson = "{" + updatedJson
	ioutil.WriteFile("./swaggerui/postman.json", []byte(updatedJson), 0644)
	// command := "./postman2openapi ./swaggerui/postman.json"
	// cmd := exec.Command(command)
	// cmd := exec.Command("postman2openapi", "-f", "json", "./swaggerui/postman.json")
	cmd := exec.Command("p2o", "./swaggerui/postman.json", "-f", "./swaggerui/swagger.yml", "-o", "./swaggerui/options.json")

	cmd.Dir = "/workspace/postman-mockserver/"
	out, err2 := cmd.CombinedOutput()
	if err2 != nil {
		fmt.Printf("cmd.Run() failed with %s\n", err2)
	}
	// ioutil.WriteFile("./swaggerui/swagger.yml", out, 0644)
	json.Unmarshal([]byte(responseJson), &postmanCollection)
	return postmanCollection, err
}

func getCollections() (Collections, error) {
	responseJson, err := getFromPostman("/collections")
	if err != nil {
		return Collections{}, err
	}

	var postmanCollections = Collections{}
	json.Unmarshal([]byte(responseJson), &postmanCollections)
	if len(postmanCollections.Collections) == 0 {
		return Collections{}, fmt.Errorf("no collections found")
	}
	return postmanCollections, nil
}

func GetMocksFromPostman() (map[string]Mock, error) {
	log.Debug().Msg("load single collection from postman...")
	collection, err := getCollection(viper.GetString("postman.collectionId"))
	mocks = make(map[string]Mock)
	if err != nil {
		log.Error().Msg("error get mock for collection " + viper.GetString("postman.collectionId") + " this collection would be skipped")
		log.Error().Msg(err.Error())
	}
	for i := 0; i < len(collection.Collection.Item); i++ {
		requests := getAllRequest(collection.Collection.Item[i], 0)
		mocks = appendMap(mocks, requests)
		// Print item name

		// log.Trace().Msg(fmt.Sprintf("%v", mocks))
		// keys := make([]string, len(mocks))

		// i := 0
		// for k1 := range mocks {
		// 	keys[i] = k1
		// 	fmt.Println(k1)
		// 	i++
		// }
		// log.Debug().Msg("[" +  "] " + collection.Collection.Item[i].Name)
	}
	return mocks, nil
}

func isIdInList(list []string, id string) bool {
	for _, v := range list {
		if strings.ToLower(v) == id {
			return true
		}
	}
	return false
}

func getFromPostman(path string) (string, error) {
	return requestPostman(path, "GET", nil)
}

func requestPostman(path string, method string, body io.Reader) (string, error) {
	log.Debug().Msg("send request to postman for " + path + " ...")
	var client = new(Client)
	client.httpClient = &http.Client{}
	fullUrl, err := url.Parse(viper.GetString("postman.url"))
	if err != nil {
		return "", err
	}
	if fullUrl.Host == "" {
		return "", fmt.Errorf("host not available, please check your configuration")
	}
	client.BaseURL = fullUrl
	if viper.GetString("postman.token") == "" {
		return "", fmt.Errorf("postman token is not present in config, please check config file")
	}
	request, err := http.NewRequest(method, fullUrl.String()+path, body)
	// log.Debug().Msg("request to postman: " + fullUrl.String() + path)
	request.Header.Set("Accept", "application/json")
	// request.Header.Set("X-Api-Key", viper.GetString("postman.token"))
	response, err := client.httpClient.Do(request)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request to postman failed, " + response.Status)
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("request to postman failed, can not read body")
	}
	bodyString := string(bodyBytes)
	log.Trace().Msg("Body get from postman: " + TruncateString(bodyString, 100))
	return bodyString, err
}
