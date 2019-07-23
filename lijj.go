package main

import (
	"fmt"
	"net/http"
)

func main() {
	




}


func test () {
	mp := map[string]string{}
	
	mp["lijj"] = "lijj"
	mp["ali"] = "baba"
	
	str, _ := json.Marshal(mp)
	fmt.Println(string(str))
	
	mpres := map[string]string{}
	json.Unmarshal(str, &mpres)
	fmt.Println(mpres)
}