package main

import (
    "bufio"
    "io"
    "log"
    "os"
)

func search_for_efergy_preamble(data []byte) {
    log.Printf("%v", cap(data))

    int negative_preamble_count=0;
    int positive_preamble_count=0;
    prvsamp = 0;
    while ( !feof(stdin) ) {
        int cursamp  = (int16_t) (fgetc(stdin) | fgetc(stdin)<<8);
        // Check for preamble
        if ((prvsamp >= analysis_wavecenter) && (cursamp >= analysis_wavecenter)) {
            positive_preamble_count++;
        } else if ((prvsamp < analysis_wavecenter) && (cursamp < analysis_wavecenter)) {
            negative_preamble_count++;
        } else if ((prvsamp >= analysis_wavecenter) && (cursamp < analysis_wavecenter)) {
            if ((positive_preamble_count > MIN_POSITIVE_PREAMBLE_SAMPLES) &&
                (negative_preamble_count > MIN_NEGATIVE_PREAMBLE_SAMPLES))
                break;
            negative_preamble_count=0;
        } else if ((prvsamp < analysis_wavecenter) && (cursamp >= analysis_wavecenter)) {
            if ((positive_preamble_count > MIN_POSITIVE_PREAMBLE_SAMPLES) &&
                (negative_preamble_count > MIN_NEGATIVE_PREAMBLE_SAMPLES))
                break;
            positive_preamble_count=0;
        }
        prvsamp = cursamp;
    } // end of find preamble while loop

}

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
