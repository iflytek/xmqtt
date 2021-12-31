package main

import "fmt"

func test() {}

func main() {
	memTopic := NewMemProvider()

	topic := []byte("/test/abc/123/456")
	topic2 := []byte("/test/abc/123/457")
	topic3 := []byte("/test/abc/123/458")
	memTopic.Subscribe(topic, 1, test)
	memTopic.Subscribe(topic2, 1, test)
	memTopic.Subscribe(topic3, 1, test)

	temp := memTopic.sroot.snodes
	for {
		if len(temp) == 0 {
			break
		}
		for k, v := range temp {
			temp = v.snodes
			fmt.Println(k, v)
		}
	}
}
