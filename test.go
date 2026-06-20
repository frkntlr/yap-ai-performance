package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	var v interface{}
	fmt.Println(json.Unmarshal([]byte("A{}"), &v))
	fmt.Println(json.Unmarshal([]byte("\x1b[31m{}"), &v))
}
