# Dashboard
The next generation of FoF Services integrated with Discord

## Features
* role assignemnt
* temporary channels
* member stream notification
* events 

# V1
 Currently in development to decouple from Slack and move to Discord
 
 ## Deployment

 This repo is integrated with Codeship for deployment. Merges and commits to  `master` and `dev` branches will trigger Codeship to build and push new images to quay.io/fofgaming/dashboard with the respective branch name as a tag (e.g., `quay.io/fofgaming/dashboard:master`)