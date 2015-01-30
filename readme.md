# GoBox

## Setup
 - Must have `GOBOX_AWS_ACCESS_KEY_ID` and `GOBOX_AWS_SECRET_ACCESS_KEY` set for aws client. 

## Api

#### Server
Endpoints:

##### /meta:
Recieves a piece of client metadata. Logs the request, and returns if the file needs to be uploaded to s3. 
```go
resp, err := http.Post("/meta", "application/json", jsonMetadataBytes)
// resp.Body #=> true
```

##### /upload:
Takes a file, computes a base256 hash value, uploads the file.  Returns any errors or sucess. 

##### /download:
Takes a file hash. (Confirms this user has access to that hash?) Passes the file from s3. 

##### /clients:
Lists clients that this user has to check. 

##### /client/1?lastCheck=dateTime:
Returns metadata for client by id after specified time. 

>**Notes:**
>Does this work? Is this enough information for the filesystem to synch? 
>How do we store user data on the server? Are there any security issues? What if a new user wants to download all files? 

>The manifests are merely a log of all files changes. Can we combine the user manifests to create a server side record of the files that exist? Then show this to the user? Show this to new user clients?

>How does the client parse the metadata and correctly download the right files?

>Right now we're using lines of json in a file. How should we be storing it instead? Database? Would a database allow us to more easily create "computed" manifests?

>Can we configure the meta endpoint to accept more than 1 piece of meta information?


## Resources
 - https://blogs.dropbox.com/tech/2014/07/streaming-file-synchronization/