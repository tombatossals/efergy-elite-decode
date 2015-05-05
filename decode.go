package main

import (
    "bufio"
    "io"
    "time"
    "fmt"
    "os"
    "strconv"
    "github.com/garyburd/redigo/redis"
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
				wrap_count++
			}
			space_count=0
			pulse_count++
		}

		if (wrap_count >= 16) {
			wrap_count=0;
		}
	}
	return pulse_count_storage, pulse_store_index;
}

func get_frame_miliwatts(bytes []byte) (int) {
	checksum := compute_checksum(bytes);
    var result float64 = 0
	if (checksum == bytes[len(bytes)-1]) {
        var bigbyte = bytes[5] >> 4;
        offset := 0
        crange := 610
        switch bigbyte {
            case 1:
                offset = 610
                crange = 2100
            case 2:
                offset = 2700
                crange = 6300
            case 3:
                offset = 7000
                crange = 18900
        }

        var current_adc float32 = float32(offset) + float32(crange)*float32(bytes[7])/float32(256)
        //var divisor = float64(90000)
        //result = float64(current_adc) * VOLTAGE / divisor
        result = float64(current_adc)
    }

    return int(result)
}


func decode_bytes_from_pulse_counts(pulse_store [SAMPLES_SIZE]int, pulse_store_index int) ([]byte) {
	dbit := 0
	bitpos := 0
	var bytedata byte = 0
    bytes := make([]byte, 0, FRAMEBYTECOUNT)
	for i:=0; i<pulse_store_index; i++ {
		if (pulse_store[i] > MINLOWBITS) {
			dbit++;
			bitpos++;
			bytedata = bytedata << 1;
			if (pulse_store[i] > MINHIGHBITS) {
				bytedata = bytedata | 0x1;
            }

			if (bitpos > 7) {
				bytes = append(bytes, bytedata);
				bytedata = 0;
				bitpos = 0;
				if (len(bytes) == FRAMEBYTECOUNT) {
					return bytes;
				}
			}
		}
	}
	return bytes;
}

func compute_checksum(data []byte) (byte) {
	var tbyte byte = 0x00;
	for i:=0; i<len(data)-1; i++ {
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

func analyze_efergy_message(data []byte, index int, samples []int16, analysis_wavecenter int16) ([]byte) {
    pulse_count_storage, pulse_store_index := generate_pulse_count_array(samples, analysis_wavecenter)
	bytearray := decode_bytes_from_pulse_counts(pulse_count_storage, pulse_store_index)

    return bytearray;
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

func redis_connect() (redis.Conn) {
    c, err := redis.Dial("tcp", "localhost:6379")
    if err != nil {
        panic(err)
    }
    return c
}

func main() {
    r := bufio.NewReader(os.Stdin)
    binary_data := make([]byte, 0, 4*1024)
    var state State  = SEARCHING_PREAMBLE;
    preambleState := PreambleState{}
    samples := make([]int16, 0, SAMPLES_SIZE)

    redis_conn := redis_connect()

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
                    bytearray := analyze_efergy_message(binary_data[:bytes_read], index, samples, analysis_wavecenter)
                    mwatts := get_frame_miliwatts(bytearray);
                    fmt.Printf("%x %d\n", bytearray, mwatts)
                    redis_conn.Do("SET", "watts:" + time.Now().Format("20060102150405"), strconv.Itoa(mwatts))
                    state = SEARCHING_PREAMBLE
                    samples = make([]int16, 0, SAMPLES_SIZE)
            }

            if (index >= len(binary_data)) {
                break;
            }
        }

        binary_data = binary_data[:bytes_read]
    }
    defer redis_conn.Close()

}
