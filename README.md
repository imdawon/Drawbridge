# Drawbridge
![Drawbridge Logo](./drawbridge_logo.jpg)

[Quick Start Guide: Protect your self-hosted servers with Drawbridge in 5 minutes or less](https://github.com/dhens/Drawbridge/wiki/Latest-Quick%E2%80%90Start-Guide-for-Drawbridge-and-Emissary-%E2%80%90-Minecraft-Server)

## The state of self-hosting 
Self-hosting is a nightmare. If you're naive, you blow a hole in your home router to allow access to whatever resource you want to have accessible via the internet. 

If you're *"smart"*, you let some other service handle the ingress for you, most likely allowing for traffic inspection and mad metadata slurp-age by said service. 
Even if there's none of that, it doesn't really feel like you're sticking it to the man when you have to rely on a service to keep your self-hosted applications secure.

## What to do about it
Emissary and Drawbridge solve this problem. 

Drawbridge is a reverse proxy which only allow machines running [Emissary desktop client](https://github.com/dhens/Emissary-Daemon) to access your self-hosted services.

Add Emissary to as many of your machines as you want via the Emissary Bundles feature, expose the Drawbridge reverse proxy server port on your router, _instead_ of directly exposong your insecure web application or "movie server", and bam: your service is only accessible from authorized Emissary clients.

[Click here to quickly set up Drawbridge and Emissary](https://github.com/dhens/Drawbridge/wiki/Latest-Quick%E2%80%90Start-Guide-for-Drawbridge-and-Emissary-%E2%80%90-Minecraft-Server)

## Example Use-Case

#### HTTP & TCP / UDP Protected Services
Creating a Protected Service in the Drawbridge dashboard creates a connection between Drawbridge and the service you want to access remotely.

A Protected Service can be any networked application listening on a given port, like a Minecraft Server or an HTTP server (currently limited to TCP).

You can then access this Protected Service by connecting to your Drawbridge server through the Emissary client. Emissary will list each service available once connected to Drawbridge and list their IP or domain names to be able to access them.


## Project Goals
The goal of the Emissary / Drawbridge solution is rapidly and easily exposing a self-hosted service to the internet for access by authorized clients.

While we want simplicity out of the box, that is not to say that you cannot enforce stricter policies for required clients. More features in the future will support additional identity requirements, but will require an admin to conduct additioal configuration of Drawbridge for such services.
~~**Note**: Currently, to ensure a high level of security, each initial connection to a Drawbridge server must be Accepted or Denied by the Drawbridge admin.~~ (not yet implemented)

To accomplish this, the following needs to be true:
- No requirement to configure TLS certificates for the Drawbridge server if using a domain name.
- Drawbridge and Emissary _only_ need eachother in order to provide all features described in this document.
- No hacky shenangians or end-user technical knowledge to verify a secure session has been created e.g checking certificate hashes or an Emissary user needing to manually configure their host machine.

## Authentication Process 
- A Drawbridge server is set up and configured to be accessible from port 3100 on an internet-facing IP address.
- An Emissary user runs the Emissary Bundle created by the Drawbridge admin to connect to Drawbridge.
- Emissary will then:
  - Create a TCP connection to Drawbridge.
  - Present its mTLS certificate to Drawbridge for the handshake process.
  - If successful, Emissary will send a message over the TCP connection to get all Protected Services available from Drawbridge.

  ### mTLS
  The Drawbridge server will create a Root CA and sign all client mTLS keys.
  
  By default, one mTLS key is shipped in the `certs` folder alongside the Emissary client "Download Bundled Emissary client" (talk_to_drawbridge.key) from the main page in the Drawbridge Dash.
  talk_to_drawbridge.key allows an Emissary client to communicate with the Drawbridge server, but they _still_ have to pass the configured Drawbridge policy to access resources protected by Drawbridge.
