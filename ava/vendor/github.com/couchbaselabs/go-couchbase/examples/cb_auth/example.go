package main

import (
	"flag"
	"github.com/couchbase/cbauth"
	"github.com/couchbase/go-couchbase"
	"log"
	"net/url"
)

var serverURL = flag.String("serverURL", "http://localhost:9000",
	"couchbase server URL")
var poolName = flag.String("poolName", "default",
	"pool name")
var bucketName = flag.String("bucketName", "default",
	"bucket name")
var authUser = flag.String("authUser", "",
	"auth user name (probably same as bucketName)")
var authPswd = flag.String("authPswd", "",
	"auth password")

func main() {

	flag.Parse()
	/*
	   NOTE. This example requires the following environment variables to be set.

	   NS_SERVER_CBAUTH_URL
	   NS_SERVER_CBAUTH_USER
	   NS_SERVER_CBAUTH_PWD

	   e.g

	   NS_SERVER_CBAUTH_URL="http://localhost:9000/_cbauth"
	   NS_SERVER_CBAUTH_USER="Administrator"
	   NS_SERVER_CBAUTH_PWD="asdasd"

	*/

	url, err := url.Parse(*serverURL)
	if err != nil {
		log.Printf("Failed to parse url %v", err)
		return
	}

	hostPort := url.Host

	user, bucket_password, err := cbauth.GetHTTPServiceAuth(hostPort)
	if err != nil {
		log.Printf("Failed %v", err)
		return
	}

	log.Printf(" HTTP Servce username %s password %s", user, bucket_password)

	client, err := couchbase.ConnectWithAuthCreds(*serverURL, user, bucket_password)
	if err != nil {
		log.Printf("Connect failed %v", err)
		return
	}

	cbpool, err := client.GetPool("default")
	if err != nil {
		log.Printf("Failed to connect to default pool %v", err)
		return
	}

	mUser, mPassword, err := cbauth.GetMemcachedServiceAuth(hostPort)
	if err != nil {
		log.Printf(" failed %v", err)
		return
	}

	var cbbucket *couchbase.Bucket
	cbbucket, err = cbpool.GetBucketWithAuth(*bucketName, mUser, mPassword)

	if err != nil {
		log.Printf("Failed to connect to bucket %v", err)
		return
	}

	log.Printf(" Bucket name %s Bucket %v", *bucketName, cbbucket)

	err = cbbucket.Set("k1", 5, "value")
	if err != nil {
		log.Printf("set failed error %v", err)
		return
	}

	if *authUser != "" {
		creds, err := cbauth.Auth(*authUser, *authPswd)
		if err != nil {
			log.Printf(" failed %v", err)
			return
		}

		canAccess, err := creds.CanAccessBucket(*bucketName)
		if err != nil {
			log.Printf(" can't access bucket %v", err)
		}

		log.Printf(" results canaccess %v bucket %v", canAccess, *bucketName)

		canRead, err := creds.CanReadBucket(*bucketName)
		if err != nil {
			log.Printf(" can't read bucket %v", err)
		}

		log.Printf(" results canread %v bucket %v", canRead, *bucketName)

		canDDL, err := creds.CanDDLBucket(*bucketName)
		if err != nil {
			log.Printf(" can't DDL bucket %v", err)
		}

		log.Printf(" results canDDL %v bucket %v", canDDL, *bucketName)
	}

}
