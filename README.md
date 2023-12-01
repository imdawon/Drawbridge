# Drawbridge
![Drawbridge Logo](./drawbridge_logo.jpg)

A reverse proxy with configurable authentication and attestation requirements to allow machines running [Emissary desktop client](https://github.com/dhens/Emissary) to access protected services.

Self-hosting is a nightmare. If you're naive, you blow a hole in your home router to allow access to whatever resource you want to have accessible via the internet. 

If you're *"smart"*, you let some other service handle the ingress for you, most likely allowing for traffic inspection and mad metadata slurp-age by said service. 
Even if there's none of that, it doesn't really feel like you're sticking it to the man when you have to rely on a service to keep your self-hosted applications secure.

Emissary and Drawbridge solve this problem. Add Emissary to as many of your machines as you want, expose the Drawbridge reverse proxy server with required authentication details, _instead_ of your insecure web application or "movie server", and bam: your service is only accessible from Emissary clients.

## How to set up as quickly as possible
### Setup Drawbridge
1. Download or build the latest version of Drawbridge
2. Run the Drawbridge binary and follow it's instructions to access the Dash.
3. Add the details for the service you want to protect in the Dash.
4. Configure your Drawbridge Policy that Emissary clients have to pass to access your service.
5. Click "Download Bundled Emissary client" and set the downloaded zip file aside for next steps.
### Set up an Emissary client
6. Transfer Emissary client from step 5 onto client machine (the one you want to access your service from)
7. Unzip the Emissary_xxxxx.zip file and run Emissary.exe
8. Enter your server IP or hostname into the connection dialog.
9. If the Emissary client reports the proper information to the entered Drawbridge server, you should see the UI turn green and indicte that you are now connected to Drawbridge!
  
  If you encountered an error, ensure your machine matches your policy you configured in step 4. ***If things still aren't working, please refer to our Troubleshooting guide.***

## Goals
The goal of the Emissary / Drawbridge solution is rapidly and easily exposing a self-hosted service to the internet for access by authorized clients.

While we want simplicity out of the box, that is not to say that you cannot enforce stricter policies for required clients. More features in the future will support additional identity requirements, but will require an admin to conduct additioal configuration of Drawbridge for such services.
**Note**: Currently, to ensure a high level of security, each initial connection to a Drawbridge server must be Accepted or Denied by the Drawbridge admin.

To accomplish this, the following needs to be true:
- No requirement to configure TLS certificates for the Drawbridge server if using a domain name.
- Drawbridge and Emissary _only_ need eachother in order to provide all features described in this document.
- No hacky shenangians or end-user technical knowledge to verify a secure session has been created e.g checking certificate hashes or an Emissary user needing to manually configure their host machine.

  
## Authentication Process 
- A Drawbridge server is set up and configured to be accessible from port 443 on an internet-facing IP address.
- For easiest deployment of Emissary clients, an Emissary 
- An Emissary user enters the IP or the URI (https://drawbridge.myserver.com) into their client.
  - Emissary will make an http request to fetch the Drawbridge policy
  - Emissary will gather the information required by the policy and send it back via an http request
  - Drawbridge will either authorize the device and return with an mTLS certificate, or an error, saying Emissary, in it's current confguration, is not authorized to access the Drawbridge resources.

  ### mTLS
  The Drawbridge server will create a Root CA and sign all client mTLS keys.
  
  By default, one mTLS key is shipped in the `certs` folder alongside the Emissary client "Download Bundled Emissary client" (talk_to_drawbridge.key) from the main page in the Drawbridge Dash.
  talk_to_drawbridge.key allows an Emissary client to communicate with the Drawbridge server, but they _still_ have to pass the configured Drawbridge policy to access resources protected by Drawbridge.
