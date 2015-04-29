var binary = require('binary');

const MIN_POSITIVE_PREAMBLE_SAMPLES = 40; /* Number of positive samples in  an efergy  preamble */
const  MIN_NEGATIVE_PREAMBLE_SAMPLES = 40; /* Number of negative samples for a valid preamble  */


var analysis_wavecenter = 0;
var negative_preamble_count=0;
var positive_preamble_count=0;
var previous_sample = 0;

var ws = binary().loop(function(end, vars) {
    this.word8s('x').tap(function(vars) {
        var current_sample = vars.x | vars.x << 8;
        console.log(new Date(), current_sample, previous_sample, positive_preamble_count, negative_preamble_count);
        if ((previous_sample >= analysis_wavecenter) && (current_sample >= analysis_wavecenter)) {
            positive_preamble_count++;
        } else if ((previous_sample < analysis_wavecenter) && (current_sample < analysis_wavecenter)) {
            negative_preamble_count++;
        } else if ((previous_sample >= analysis_wavecenter) && (current_sample < analysis_wavecenter)) {
            if ((positive_preamble_count > MIN_POSITIVE_PREAMBLE_SAMPLES) &&
                (negative_preamble_count > MIN_NEGATIVE_PREAMBLE_SAMPLES)) {
                    console.log('found');
                    end();
                }
            negative_preamble_count=0;
        } else if ((previous_sample < analysis_wavecenter) && (current_sample >= analysis_wavecenter)) {
            if ((positive_preamble_count > MIN_POSITIVE_PREAMBLE_SAMPLES) &&
                (negative_preamble_count > MIN_NEGATIVE_PREAMBLE_SAMPLES)) {
                    console.log('found');
                    end();
                }
            positive_preamble_count=0;
        }
        previous_sample = current_sample;

        if (vars.x === null) {
            console.log('eof');
            end();
            return;
        }
    });
});


process.stdin.pipe(ws);
process.stdin.resume();
