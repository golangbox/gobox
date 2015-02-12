# GoBox

GoBox is a Dropbox clone written in Go. 

The design is a client/server architecture using HTTP endpoints to communicate file change events between clients. A global journal of all file changes is kept on the server in a Postgres database.

Local file changes are hashed and sent to the server, which discerns whether or not it has a file under that hash already. If not, the client uploads the file to the server, where the file is hashed to check for integrity, and if valid uploaded to an Amazon S3 instance. All other clients are alerted that a change has been made through a UDP socket, and then the other clients request the necessary changes through an HTTP endpoint. Clients then get the necessary changes directly from the Amazon S3 instance through an S3 signed URL.

## Notes
 - Must have `GOBOX_AWS_ACCESS_KEY_ID` and `GOBOX_AWS_SECRET_ACCESS_KEY` set for aws client. 
 - os.FileMode struct has all the information me need to handle files. symlink, permission, directory, etc....
 - https://blogs.dropbox.com/tech/2014/07/streaming-file-synchronization/
 - https://www.youtube.com/watch?v=PE4gwstWhmc

## Api

#### Server Endpoints:

##### POST: /file-actions/

##### POST: /upload/

##### POST: /download/

##### POST: /clients/

## Resources
