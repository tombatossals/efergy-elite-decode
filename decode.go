package main

import (
    "bufio"
    "io"
    "math"
    "fmt"
    "time"
    "os"
)

type State int
const (
    SEARCHING_PREAMBLE State = 1 + iota
    GETTING_SAMPLES
    ANALYZING_MESSAGE
)

type PreambleState struct {
    negative_preamble_count int16
    positive_preamble_count int16
    previous_sample int16
}


const VOLTAGE = 220
const MINLOWBITS = 3
const MINHIGHBITS = 9
const MIN_POSITIVE_PREAMBLE_SAMPLES = 40
const MIN_NEGATIVE_PREAMBLE_SAMPLES = 40
const APPROX_SAMPLES_PER_BIT =19
const FRAMEBYTECOUNT = 15
const FRAMEBITCOUNT = FRAMEBYTECOUNT*8
const SAMPLES_SIZE = FRAMEBITCOUNT*APPROX_SAMPLES_PER_BIT

var debug_level int = 4


func generate_pulse_count_array(samples []int16, analysis_wavecenter int16) ([SAMPLES_SIZE]int, int) {
	var store_positive_pulses bool = (samples[2] < analysis_wavecenter)
    var pulse_count_storage [SAMPLES_SIZE]int

    fmt.Printf("\nPulse stream for this frame (P-Consecutive samples > center, N-Consecutive samples < center)\n")

	wrap_count := 0
	pulse_count := 0
	space_count := 0
	pulse_store_index := 0

	for i:=0;i<len(samples);i++ {
		samplec := samples[i] - analysis_wavecenter
		if (samplec < 0) {
			if (pulse_count > 0) {
				if (store_positive_pulses) {
				  pulse_count_storage[pulse_store_index] = pulse_count
                  pulse_store_index++
                }

                fmt.Printf("%2dP ", pulse_count)
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
                fmt.Printf("%2dN ", space_count)
				wrap_count++
			}
			space_count=0
			pulse_count++
		}

		if (wrap_count >= 16) {
            fmt.Printf("\n")
			wrap_count=0;
		}
	}
    fmt.Printf("\n\n")

	return pulse_count_storage, pulse_store_index;
}

func display_frame_data(debug_level int, bytes [FRAMEBYTECOUNT]byte, bytecount int) {
	fmt.Printf("%x", bytes)
    var data_ok_str string = ""
	var checksum byte = 0

	checksum = compute_checksum(bytes, bytecount);
	if (checksum == bytes[bytecount-1]) {
		data_ok_str = "chksum ok"
    }

	var current_adc float32 = float32(bytes[4]) * 256 + float32(bytes[5])
	var result float64 = float64(VOLTAGE*current_adc) / 32768 / math.Pow(2, float64(bytes[6]))
	if (debug_level > 0) {
		if (debug_level == 1) {
			fmt.Printf("%s ", time.Now());
        }

		for i:=0;i<bytecount;i++ {
			fmt.Printf("%02x ",bytes[i])
        }

		if (data_ok_str != "") {
			fmt.Printf("%s", data_ok_str)
		} else {
			checksum = compute_checksum(bytes, bytecount)
            fmt.Printf(" cksum: %02x ",checksum)
		}

		if (result < 100) {
            fmt.Printf("  kW: %4.3f\n", result)
		} else {
            fmt.Printf("  kW: <out of range>\n");
			if (data_ok_str !=  "") {
                fmt.Printf("*For Efergy True Power Moniter (TPM), set VOLTAGE=1 before compiling\n")
            }
        }

    } else if (data_ok_str != "") {
		fmt.Printf("%s,%f\n",time.Now(),result);
	} else {
        fmt.Printf("Checksum/CEC Error.  Enable debug output with -d option\n")
    }
}


func decode_bytes_from_pulse_counts(pulse_store [SAMPLES_SIZE]int, pulse_store_index int) ([FRAMEBYTECOUNT]byte, int) {
	dbit := 0
	bitpos := 0
	var bytedata byte = 0
    var bytes [FRAMEBYTECOUNT]byte
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
					return bytes, bytecount;
				}
			}
		}
	}
	return bytes, bytecount;
}

func compute_checksum(data [FRAMEBYTECOUNT]byte, bytecount int) (byte) {
	var tbyte byte = 0x00;
	for i:=0;i<(bytecount-1);i++ {
	  tbyte += data[i];
	}

	return tbyte;
}

func calculate_wave_center(samples []int16) (int16) {
	var avg_neg int64 = 0
	var avg_pos int64 = 0
	var pos_count int64 =0;
	var neg_count int64 =0;

	for i:=0;i<len(samples);i++ {
		if (samples[i] >=0) {
			avg_pos += int64(samples[i]);
			pos_count++;
		} else {
			avg_neg += int64(samples[i]);
			neg_count++;
		}
    }

	if (pos_count!=0) {
		avg_pos /= pos_count;
    }

	if (neg_count!=0) {
		avg_neg /= neg_count;
    }

	return int16(avg_neg + ((avg_pos-avg_neg)/2))
}

func analyze_efergy_message(data []byte, index int, samples []int16, analysis_wavecenter int16) {
    pulse_count_storage, pulse_store_index := generate_pulse_count_array(samples, analysis_wavecenter)
	bytearray, bytecount := decode_bytes_from_pulse_counts(pulse_count_storage, pulse_store_index)
	display_frame_data(debug_level, bytearray, bytecount);

	if (debug_level>1) {
        fmt.Printf("\n")
    }
}

func get_samples(data []byte, index int, samples []int16) (State, []int16, int) {
    for index < len(data)-1 {
        a := uint(data[index])
        b := uint(data[index+1])
        samples = append(samples, int16(a | b << 8))
        if (len(samples) == SAMPLES_SIZE) {
            return ANALYZING_MESSAGE, samples, index
        }

        index += 2
    }

    return GETTING_SAMPLES, samples, index
}

func search_preamble(data []byte, index int, preambleState PreambleState, analysis_wavecenter int16) (State, int, PreambleState) {

    for index < len(data)-1 {
        a := uint(data[index])
        b := uint(data[index+1])

        current_sample := int16(a | b << 8)
        if ((preambleState.previous_sample >= analysis_wavecenter) && (current_sample >= analysis_wavecenter)) {
            preambleState.positive_preamble_count++;
        } else if ((preambleState.previous_sample < analysis_wavecenter) && (current_sample < analysis_wavecenter)) {
            preambleState.negative_preamble_count++;
        } else if ((preambleState.previous_sample >= analysis_wavecenter) && (current_sample < analysis_wavecenter)) {
            if ((preambleState.positive_preamble_count > MIN_POSITIVE_PREAMBLE_SAMPLES) &&
                (preambleState.negative_preamble_count > MIN_NEGATIVE_PREAMBLE_SAMPLES)) {
                    index += 2
                    return GETTING_SAMPLES, index, preambleState
            }
            preambleState.negative_preamble_count = 0;
        } else if ((preambleState.previous_sample < analysis_wavecenter) && (current_sample >= analysis_wavecenter)) {
            if ((preambleState.positive_preamble_count > MIN_POSITIVE_PREAMBLE_SAMPLES) &&
                (preambleState.negative_preamble_count > MIN_NEGATIVE_PREAMBLE_SAMPLES)) {
                    index += 2
                    return GETTING_SAMPLES, index, preambleState
            }
            preambleState.positive_preamble_count = 0;
        }
        preambleState.previous_sample = current_sample;
        index += 2
    }

    return SEARCHING_PREAMBLE, index, preambleState
}

func main() {
    r := bufio.NewReader(os.Stdin)
    binary_data := make([]byte, 0, 4*1024)
    var state State  = SEARCHING_PREAMBLE;
    preambleState := PreambleState{}
    samples := make([]int16, 0, SAMPLES_SIZE)

    var analysis_wavecenter int16 = 0

    for {
        bytes_read, err := r.Read(binary_data[:cap(binary_data)])
        if err == io.EOF {
            break
        }

        index := 0
        for {
            switch state {
                case SEARCHING_PREAMBLE:
                    state, index, preambleState = search_preamble(binary_data[:bytes_read], index, preambleState, analysis_wavecenter)
                case GETTING_SAMPLES:
                    preambleState = PreambleState{}
                    state, samples, index = get_samples(binary_data[:bytes_read], index, samples)
                case ANALYZING_MESSAGE:
                    analysis_wavecenter = calculate_wave_center(samples)
                    analyze_efergy_message(binary_data[:bytes_read], index, samples, analysis_wavecenter)
                    os.Exit(1)
            }

            if (index >= len(binary_data)) {
                break;
            }
        }

        binary_data = binary_data[:bytes_read]
    }
}
