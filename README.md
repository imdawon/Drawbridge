# Drawbridge
![Drawbridge Logo](./drawbridge_logo.jpg)

A reverse proxy with configurable authentication and attestation requirements to allow [Emissary desktop client](https://github.com/dhens/Emissary) machines to access resources beyond the proxy.

Self-hosting is a nightmare. If you're naive, you blow a hole in your home router to allow access to whatever resource you want to have accessible via the internet. 

If you're *"smart"*, you let some other service handle the ingress for you, most likely allowing for traffic inspection and mad metadata slurp-age by said service. 
Even if there's none of that, it doesn't really feel like you're sticking it to the man when you have to rely on a service to keep your self-hosted applications secure.

Emissary and Drawbridge solve this problem. Add Emissary to as many of your machines as you want, expose the Drawbridge reverse proxy server with required authentication details, _instead_ of your insecure web application or "movie server", and bam: your service is only accessible from Emissary clients.

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
  - This will open a websocket connection 
