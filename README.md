# VPNExiter

### What?

If you like using a VPN provider on your router so that multiple devices
on your network can take advantage of the VPN, you may find it hard to
manage your VPN connections and choose different exit gateways based
on performance or geo-location requirements.

VPNExiter is a GoLang service which runs on your router and makes it
convienent to manage your VPN connection.

### Features

 * Easily switch between different VPN providers or exit points
 * Easily benchmark VPN tunnels via speedtest.net

### Supported Platforms

#### On Device

Development was done on the Ubiquiti USG and EdgeRouter Lite-3.  These were
picked due to their low cost and relatively limited hardware specs (dual-core
500Mhz MIPS64 CPU and 512MB of RAM).  Hence, you should be able to run
VPNExiter on any
[hardware/OS that GoLang supports](https://github.com/golang/go/wiki/MinimumRequirements).

#### Docker
You can also run VPNExiter on a computer on your network via
[Docker](https://www.docker.com).  This is a more complicated setup, since VPNExiter
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

### Speedtest.net Integration

VPNExiter supports two different integration types:

 1. Embed your own [Speedtest Custom](https://www.ookla.com/speedtest-custom) HTML5 test app.
    You can sign up for free.  This then runs the speedtest app in your browser which has a pretty
    UI, but may not be able to saturate high capacity links.
 1. Run the [Speedtest CLI](https://www.speedtest.net/apps/cli) on the same server as VPNExiter.
    Free without any sign up requirements.  Since this doesn't run in the browser,
    it may support higher throughput than the HTML5 app.  But it doesn't support all platforms
    (no MIPS64 support for Ubiquiti USG for example).  Also, it has an "ugly as sin" front end
    because I suck at web applications.


### Configuration

Configuring VPNExiter is done via a single <em>config.yaml</em> file.

 * listen:
    * http: *http port*
    * https: *https port*
    * username: *http auth username*
    * password: *http auth password using bcrypt: `htpasswd -nbB <username> <password>`*

 * speedtest_cli: *path to speedtest cli tool*
 * speedtest_url: https://*custom*.speedtestcustom.com

 * router:
    * mode: *ssh | local*
    * config_file: *ipsec.conf*
    * start_command: */usr/sbin/ipsec start*
    * stop_command: */usr/sbin/ipsec stop*
    * status_command: */usr/sbin/ipsec statusall | grep -q '1 up'*
    * host: *IP address of router (default: localhost)*
    * port: *port sshd listens on (default 22)*
    * username: *ssh username*
    * password: *ssh password*

 * vendors:
    - *vendor name 1*
    - *vendor name 2*
    - *vendor name X*

 * *vendor name 1*
    * config_template: *path to config template*
    * levels:
        - *level 1*
        - *level 2*
    * servers:
        * Level 1:
            * Level 2:
                - server 1
                - server 2
                - server 3
            * Level 2:
                - server 1
        * Level 1:
            * Level 2:
                - server 1
            * Level 2:
                - server 1
            * Level 2:
                - server 1
        * Level 1:
            * Level 2:
                - server 1
                - server 2

 * *vendor name 2*
    * config_template: *path to config template*
    * servers:
        - server 1
        - server 2
