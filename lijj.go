package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	mp := map[string]string{}
	
	mp["lijj"] = "lijj"
	mp["ali"] = "baba"
	
	str, _ := json.Marshal(mp)
	fmt.Println(string(str))
	
	mpres := map[string]string{}
	json.Unmarshal(str, &mpres)
	fmt.Println(mpres)
}
