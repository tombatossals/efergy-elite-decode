package main

import (
    "bufio"
    "io"
    "math"
    "fmt"
    "time"
    "os"
)

const VOLTAGE = 220
const MINLOWBITS = 3
const MINHIGHBITS = 9
const MIN_POSITIVE_PREAMBLE_SAMPLES = 40
const MIN_NEGATIVE_PREAMBLE_SAMPLES = 40
const APPROX_SAMPLES_PER_BIT =19
const FRAMEBYTECOUNT = 9
const FRAMEBITCOUNT = FRAMEBYTECOUNT*8
const SAMPLE_STORE_SIZE = FRAMEBITCOUNT*APPROX_SAMPLES_PER_BIT

var analysis_wavecenter int16 = 0
var negative_preamble_count int16 =0
var positive_preamble_count int16 = 0
var previous_sample int16 = 0
var sample_storage [SAMPLE_STORE_SIZE]int16
var sample_store_index int
var debug_level int = 1





func generate_pulse_count_array(display_pulse_details bool, pulse_count_storage []int) (int) {
	var store_positive_pulses bool = (sample_storage[2] < analysis_wavecenter)

	if (display_pulse_details) {
        fmt.Printf("\nPulse stream for this frame (P-Consecutive samples > center, N-Consecutive samples < center)\n")
    }

	wrap_count := 0
	pulse_count := 0
	space_count := 0
	pulse_store_index := 0
	space_store_inde := 0
	display_pulse_info := 1

	for i:=0;i<sample_store_index;i++ {
		samplec := sample_storage[i] - analysis_wavecenter
		if (samplec < 0) {
			if (pulse_count > 0) {
				if (store_positive_pulses) {
				  pulse_count_storage[pulse_store_index] = pulse_count
                  pulse_store_index++
                }

				if (display_pulse_details) {
                    fmt.Printf("%2dP ", pulse_count)
                }

				wrap_count++;
			}
			pulse_count=0;
			space_count++;

		} else {
			if (space_count > 0) {
				if (!store_positive_pulses) {
				    pulse_count_storage[pulse_store_index]=space_count
                    pulse_store_index++
                }
				if (display_pulse_details) {
                    fmt.Printf("%2dN ", space_count)
                }
				wrap_count++
			}
			space_count=0
			pulse_count++
		}

		if (wrap_count >= 16) {
			if (display_pulse_details) {
                fmt.Printf("\n")
            }
			wrap_count=0;
		}
	}
	if (display_pulse_details) {
        fmt.Printf("\n\n")
    }

	return pulse_store_index;
}

func display_frame_data(debug_level int, msg string, bytes []byte, bytecount int) {

	// Some magic here to figure out whether the message has a 1 byte checksum or 2 byte crc
	var data_ok_str byte = 0
	var checksum byte = 0
	var crc int16 = 0

	checksum = compute_checksum(bytes, bytecount);
	if (checksum == bytes[bytecount-1]) {
		data_ok_str := "chksum ok"
    }

	var current_adc float32 = float32(bytes[4]) * 256 + float32(bytes[5])
	var result float64 = float64(VOLTAGE*current_adc) / 32768 / math.Pow(2, float64(bytes[6]))
	if (debug_level > 0) {
		if (debug_level == 1) {
			fmt.Printf("%s  %s ", buffer, msg);
		} else {
            fmt.Printf("%s ", msg);
        }

		for i:=0;i<bytecount;i++ {
			fmt.Printf("%02x ",bytes[i])
        }

		if (data_ok_str != 0) {
			fmt.Printf(data_ok_str)
		} else {
			checksum = compute_checksum(bytes, bytecount)
			crc = compute_crc(bytes, bytecount)
			printf(" cksum: %02x crc16: %04x ",checksum, crc)
		}

		if (result < 100) {
			printf("  kW: %4.3f\n", result)
		} else {
			printf("  kW: <out of range>\n");
			if (data_ok_str !=  0) {
			  printf("*For Efergy True Power Moniter (TPM), set VOLTAGE=1 before compiling\n")
            }
        }

    } else if (data_ok_str != 0) {
		fmt.Printf("%s,%f\n",buffer,result);
		if (loggingok) {
		  if (LOGTYPE) {
		      fmt.Printf("%s,%f\r\n",buffer,result)
		  } else {
              fmt.Printf(fp,"%s,%f\n",buffer,result)
		  }
		  samplecount++;
		  if (samplecount==SAMPLES_TO_FLUSH) {
		    samplecount=0;
		    fflush(fp);
		  }
		}
	} else {
		printf("Checksum/CEC Error.  Enable debug output with -d option\n")
    }
}


func decode_bytes_from_pulse_counts(pulse_store []int, pulse_store_index int, bytes []byte) (int) {
	dbit := 0
	bitpos := 0
	var bytedata byte = 0
	bytecount := 0

	for i:=0;i<FRAMEBYTECOUNT;i++ {
		bytes[i]=0;
    }

	for i:=0; i<pulse_store_index; i++ {
		if (pulse_store[i] > MINLOWBITS) {
			dbit++;
			bitpos++;
			bytedata = bytedata << 1;
			if (pulse_store[i] > MINHIGHBITS) {
				bytedata = bytedata | 0x1;
            }

			if (bitpos > 7) {
				bytes[bytecount] = bytedata;
				bytedata = 0;
				bitpos = 0;
				bytecount++;
				if (bytecount == FRAMEBYTECOUNT) {
					return bytecount;
				}
			}
		}
	}

	return bytecount;
}

func compute_checksum(data []byte, bytecount int) (byte) {
	var tbyte byte = 0x00;
	for i:=0;i<(bytecount-1);i++ {
	  tbyte += data[i];
	}

	return tbyte;
}

func calculate_wave_center(sample_storage [SAMPLE_STORE_SIZE]int16, sample_store_index int) (int, int, int) {
	var avg_neg int64 = 0
	var avg_pos int64 = 0
	var pos_count int64 =0;
	var neg_count int64 =0;

	for i:=0;i<sample_store_index;i++ {
		if (sample_storage[i] >=0) {
			avg_pos += int64(sample_storage[i]);
			pos_count++;
		} else {
			avg_neg += int64(sample_storage[i]);
			neg_count++;
		}
    }

	if (pos_count!=0) {
		avg_pos /= pos_count;
    }

	if (neg_count!=0) {
		avg_neg /= neg_count;
    }

	var diff int = int(avg_neg + ((avg_pos-avg_neg)/2))
	return int(avg_pos), int(avg_neg), diff
}

func analyze_efergy_message(data []byte, index int) {
	avg_pos, avg_neg, difference := calculate_wave_center(sample_storage, sample_store_index);

	if (debug_level > 1) {
		fmt.Println("\nAnalysis of rtl_fm sample data for frame received on %s", time.Now());
        fmt.Println("     Number of Samples: %6d", sample_store_index);
        fmt.Println("    Avg. Sample Values: %6d (negative)   %6d (positive)", avg_neg, avg_pos);
        fmt.Println("           Wave Center: %6d (this frame) %6d (last frame)", difference, analysis_wavecenter);
	}

	analysis_wavecenter = int16(difference); // Use the calculated wave center from this sample to process next frame

	if (debug_level==4) { // Raw Sample Dump only in highest debug level
		wrap_count :=0
        fmt.Println("Showing raw rtl_fm sample data received between start of frame and end of frame");
		for i:=0;i<sample_store_index;i++ {
			fmt.Printf("%6d ", sample_storage[i] - analysis_wavecenter);
			wrap_count++;
			if (wrap_count >= 16) {
				fmt.Printf("\n");
				wrap_count=0;
			}
		}
		fmt.Printf("\n\n");
	}

	var display_pulse_details bool = (debug_level >= 3)
	var pulse_count_storage [SAMPLE_STORE_SIZE]int
	pulse_store_index := generate_pulse_count_array(display_pulse_details, pulse_count_storage)
	var bytearray [FRAMEBYTECOUNT]byte
	bytecount := decode_bytes_from_pulse_counts(pulse_count_storage, pulse_store_index, bytearray)
	var frame_msg string;
	if (sample_storage[2] < analysis_wavecenter) {
		frame_msg = "Msg:"
	} else {
		frame_msg = "Msg (from negative pulses):"
    }

	display_frame_data(debug_level, frame_msg, bytearray, bytecount);

	if (debug_level>1) {
        printf("\n")
    }
}

func process_efergy_frame(data []byte, index int) {
    for index < len(data)-1 {
        a := uint(data[index])
        b := uint(data[index+1])
        current_sample := int16(a | b << 8)
        fmt.Println(sample_store_index)
        sample_storage[sample_store_index] = current_sample
        if (sample_store_index < (SAMPLE_STORE_SIZE-1)) {
            sample_store_index++;
        } else {
            analyze_efergy_message(data, index);
            break;
        }

        index += 2
    }
}

func search_efergy_preamble(data []byte) {

    index := 0
    for index < len(data)-1 {
      a := uint(data[index])
      b := uint(data[index+1])

      current_sample := int16(a | b << 8)
      if ((previous_sample >= analysis_wavecenter) && (current_sample >= analysis_wavecenter)) {
          positive_preamble_count++;
      } else if ((previous_sample < analysis_wavecenter) && (current_sample < analysis_wavecenter)) {
          negative_preamble_count++;
      } else if ((previous_sample >= analysis_wavecenter) && (current_sample < analysis_wavecenter)) {
          if ((positive_preamble_count > MIN_POSITIVE_PREAMBLE_SAMPLES) &&
              (negative_preamble_count > MIN_NEGATIVE_PREAMBLE_SAMPLES)) {
                  index += 2
                  process_efergy_frame(data, index)
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
      index += 2
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
        search_efergy_preamble(buf[:n])
        buf = buf[:n]
    }
}
