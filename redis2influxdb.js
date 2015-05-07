#!/usr/bin/env node

var redis_cli = require('redis').createClient();
var Q = require('q');
var pw= require('powerpromises');
var influx = require('influx');

var influx_client = influx({
    host : 'localhost',
    username : 'watt',
    password : 'watt',
    database : 'watt'
});

var PARALLEL = 5;

function redis_keys(cad) {
    var df = Q.defer();
    redis_cli.keys(cad, function(err, reply) {
        df.resolve(reply);
    });

    return df.promise;
}


function redis_del(key) {
    var df = Q.defer();
    redis_cli.del(key, function(err, reply) {
        df.resolve();
    });

    return df.promise;
}

function influxdb_send(key, value) {
    var df = Q.defer();
    var point = { time: new Date(), watts: value };
    influxdb_client.writePoint("watt", point, function() {
        df.resolve();
    });

    return df.promise;
}

function redis_get(key) {
    var df = Q.defer();
    redis_cli.get(key, function(err, reply) {
        df.resolve(reply);
    });

    return df.promise;
}

function process_key(key) {
    var df = Q.defer();

    redis_get(key).then(function(value) {
        influxdb_send(key, value).then(function() {
	    redis_del(key).then(function() {
                console.log(key);
    	        df.resolve(result);
            });
        });
    });
    return df.promise;
}

redis_keys('watts:*').then(function(keys) {
    pw.chainPromises(process_key, keys, PARALLEL).then(function() {
        redis_cli.quit();
    });
});
