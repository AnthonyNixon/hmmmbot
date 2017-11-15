package main

import "net/http"
import "fmt"
import (
	"encoding/json"
	"encoding/base64"
	"time"
	"log"
	"io/ioutil"
	"math/rand"
	"os"
	"io"
	"golang.org/x/net/context"
	vision "cloud.google.com/go/vision/apiv1"
	"github.com/ChimeraCoder/anaconda"
	"net/url"
	"bufio"
)

type PostData struct {
	Url string `json:"url"`
	NSFW bool `json:"over_18"`
}

type Listing struct {
	Kind string `json:"kind"`
	Data struct {
		Children []struct {
			Posts PostData `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func main() {
	link := getRandomLink()
	fmt.Printf("Using link: '%s'", link)

	saveImage(link)
	label, err := detectLabels("image.jpg")
	if err != nil {
		fmt.Println(err.Error())
	}

	sendTweet(label, link)

}

func saveImage(link string) {
	// don't worry about errors
	response, e := http.Get(link)
	if e != nil {
		log.Fatal(e)
	}

	defer response.Body.Close()

	//open a file for writing
	file, err := os.Create("image.jpg")
	if err != nil {
		log.Fatal(err)
	}
	// Use io.Copy to just dump the response body to the file. This supports huge files
	_, err = io.Copy(file, response.Body)
	if err != nil {
		log.Fatal(err)
	}
	file.Close()
	fmt.Println("Success!")
}

func getRandomLink() string {
	url := "https://www.reddit.com/r/hmmm/hot/.json?count=20"

	redditClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}


	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "hmmmbot")

	res, getErr := redditClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	//fmt.Printf("%s", body)


	listing1 := Listing{}
	jsonErr := json.Unmarshal(body, &listing1)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	var urlArray []string
	//fmt.Printf("%s", listing1.Data.Children)
	for _, element := range listing1.Data.Children {
		if !element.Posts.NSFW {
			urlArray = append(urlArray, element.Posts.Url)
		}
		//fmt.Printf("%s\n", element.Posts.Url)
	}

	//fmt.Print(urlArray)
	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
	link := fmt.Sprintf("%s", urlArray[rand.Intn(len(urlArray))])
	//fmt.Println(link)
	return link
}

func detectLabels(file string) (string, error) {
	ctx := context.Background()

	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return "", err
	}

	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	image, err := vision.NewImageFromReader(f)
	if err != nil {
		return "", err
	}
	annotations, err := client.DetectLabels(ctx, image, nil, 10)
	if err != nil {
		return "", err
	}

	if len(annotations) == 0 {
		fmt.Printf("No labels found.\n")
	} else {
		fmt.Printf("Labels:\n")
		for _, annotation := range annotations {
			fmt.Printf("%s\n", annotation)
		}
	}

	return annotations[len(annotations)-1].Description, nil
}

func sendTweet(description string, imageUrl string) {
	anaconda.SetConsumerKey("")
	anaconda.SetConsumerSecret("")
	api := anaconda.NewTwitterApi("", "")

	file, err := os.Open("image.jpg")
	if err != nil {
		fmt.Println(err.Error())
	}

	defer file.Close()
	fileInfo, _ := file.Stat()
	var size int64 = fileInfo.Size()
	bytes := make([]byte, size)

	buffer := bufio.NewReader(file)
	_, err = buffer.Read(bytes)

	encodedString := base64.StdEncoding.EncodeToString(bytes)
	//fmt.Printf("Encoded String:\n%s\n", encodedString)

	media, err := api.UploadMedia(encodedString)
	if err != nil {
		fmt.Println(err.Error())
	}

	values := url.Values{}
	values.Set("media_ids", media.MediaIDString)
	api.PostTweet("\"" + description + "\"" + "\n", values)
}