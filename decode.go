package main

import (
    "bufio"
    "io"
    "fmt"
    "os"
)

const MIN_POSITIVE_PREAMBLE_SAMPLES = 40; /* Number of positive samples in  an efergy  preamble */
const  MIN_NEGATIVE_PREAMBLE_SAMPLES = 40; /* Number of negative samples for a valid preamble  */

var analysis_wavecenter int16 = 0;
var negative_preamble_count int16 =0;
var positive_preamble_count int16 = 0;
var previous_sample int16 = 0;

func search_for_efergy_preamble(data []byte) {

    for i := 0; i < len(data)-1; i += 2 {
      a := uint(data[i])
      b := uint(data[i+1])

      current_sample := int16(a | b << 8)
      fmt.Println("--", a, b, current_sample, "--")

      if ((previous_sample >= analysis_wavecenter) && (current_sample >= analysis_wavecenter)) {
          positive_preamble_count++;
      } else if ((previous_sample < analysis_wavecenter) && (current_sample < analysis_wavecenter)) {
          negative_preamble_count++;
      } else if ((previous_sample >= analysis_wavecenter) && (current_sample < analysis_wavecenter)) {
          if ((positive_preamble_count > MIN_POSITIVE_PREAMBLE_SAMPLES) &&
              (negative_preamble_count > MIN_NEGATIVE_PREAMBLE_SAMPLES)) {
                  os.Exit(1)
              }
          negative_preamble_count=0;
      } else if ((previous_sample < analysis_wavecenter) && (current_sample >= analysis_wavecenter)) {
          if ((positive_preamble_count > MIN_POSITIVE_PREAMBLE_SAMPLES) &&
              (negative_preamble_count > MIN_NEGATIVE_PREAMBLE_SAMPLES)) {
                os.Exit(1)
              }
          positive_preamble_count=0;
      }
      previous_sample = current_sample;
  }
}

func main() {
    r := bufio.NewReader(os.Stdin)
    buf := make([]byte, 0, 4*1024)
    for {
        n, err := r.Read(buf[:cap(buf)])
        if err == io.EOF {
            break
        }
        search_for_efergy_preamble(buf[:n])
        buf = buf[:n]
    }
}
