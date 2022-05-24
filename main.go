package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/datadotworld/dwapi-go/dwapi"
)

func main() {
	l := FetchDatasets()

	fmt.Println("hola", len(l))
}

func FetchDatasets() []string {
	// new client
	token := os.Getenv("DW_AUTH_TOKEN")
	dw := dwapi.NewClient(token)

	owner := "chrisbodhi"

	fmt.Println("owner is", owner)

	var toDownload []string

	ownedList, err := dw.Dataset.Owned()
	if err != nil {
		log.Fatalln(err)
	}

	for _, o := range ownedList {
		toDownload = append(toDownload, o.ID)
	}

	fmt.Println("ownedList includes", toDownload)

	// This takes way too long to run -- shelve for now, as not needed for POC
	// contribList, err := dw.Dataset.Contributing()
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// fmt.Println("got", len(contribList), "items for contrib")

	// for _, c := range contribList {
	// 	toDownload = append(toDownload, c.ID)
	// }

	// fmt.Println("contribList is at the end of", toDownload)

	downloads := 0

	for _, id := range toDownload {
		fmt.Println("getting", id)
		r, err := dw.Dataset.DownloadAndSave(owner, id, "./"+id+".zip")
		downloads += 1
		if downloads > 10 {
			fmt.Println("(Taking a breather...)")
			// Pause for 429's
			time.Sleep(60 * time.Second)
			downloads = 0
		}
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(r.Message)
	}

	return toDownload
}
