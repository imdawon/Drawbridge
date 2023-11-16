# Drawbridge
![Drawbridge Logo](./drawbridge_logo.png)

A reverse proxy with configurable authentication and attestation requirements to allow [Emissary desktop client](https://github.com/dhens/Emissary) machines to access resources beyond the proxy.

Self-hosting is a nightmare. If you're naive, you blow a hole in your home router to allow access to whatever resource you want to have accessible via the internet. If you're *"smart"*, you let some other service handle the ingress for you, most likely allowing for traffic inspection and mad metadata slurp-age by said service. Even if there's none of that, it doesn't really feel like you're sticking it to the man when you have to rely on a service to keep your self-hosted applications secure.

Emissary and Drawbridge solve this problem. Add Emissary to as many of your machines as you want, expose the Drawbridge reverse proxy server with required authentication details, _instead_ of your insecure web application or "movie server", and bam: your service is only accessible from Emissary clients.
