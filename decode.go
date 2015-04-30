package main

import (
    "bufio"
    "io"
    "log"
    "os"
)


func main() {
    reader := bufio.NewReader(os.Stdin)
    for {
        data := make([]byte, 4<<20) // Read 4MB at a time
        _ , err := reader.Read(data)
        search_for_efergy_preamble(data)
        if err == io.EOF {
            break
        } else if err != nil {
            log.Fatalf("Problems reading from input: %s", err)
        }
    }
}
