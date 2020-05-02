# VPN Exiter

### What?

If you like using a VPN provider on your router so that multiple devices
on your network can take advantage of the VPN, you may find it hard to 
manage your VPN connections and choose different exit gateways based
on performance or geo-location requirements.

VPN Exiter is a GoLang service which runs on your router and makes it
convienent to manage your VPN connection.

### Features

 * Easily switch between different VPN providers or exit points
 * Easily benchmark VPN tunnels

### Supported Platforms

#### On Device

Development was done on the Ubiquiti USG and EdgeRouter Lite-3.  These were
picked due to their low cost and relatively limited hardware specs (dual-core
500Mhz MIPS64 CPU and 512MB of RAM).  Hence, you should be able to run 
VPN Exiter on any 
[hardware/OS that GoLang supports](https://github.com/golang/go/wiki/MinimumRequirements).

Note that certain functions (like [Speedtest](https://www.speedtest.net) integration
may not be available on certain platforms.  See the [speedtest website](https://www.speedtest.net/apps/cli)
for a list of supported platforms.

#### Docker
You can also run VPN Exiter on a computer on your network via 
[Docker](https://www.docker.com).  This is a more complicated setup, since VPN Exiter
has to ssh into your router to reconfigure it, but may be preferable for some
use cases.


### Supported VPNs

Tested VPNs are:

 - StrongSwan
 - OpenVPN

However, any VPN solution that meets the following requirements should be
possible:

 1. There are commands to start, stop and get the status of the service
 1. A single file contains the necessary configuration information to switch
    between VPN exit servers
