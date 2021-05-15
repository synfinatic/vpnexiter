# VPNExiter

### What?

If you like using a VPN provider on your router so that multiple devices
on your network can take advantage of the VPN, you may find it hard to
manage your VPN connections and choose different exit gateways based
on performance or geo-location requirements.

VPNExiter is a GoLang service which runs on your router (or other device)
and makes it convienent to manage your VPN connection.

### Project Status

**VPNExiter is no longer being developed at this time as I've ended up moving to
another solution and no longer need this.**

VPNExiter is early beta.  Right now I have it working for a single use case
to the point where it is useful.  But I'm sure there are bugs and lots of missing
features before I consider it "good enough".

Feel free to download and give it a try, but right now I'd consider it only
usable by advanced users.  Bug reports and pull requests are welcome!

### Features

 * Easily switch between different VPN providers or exit points
 * Easily benchmark VPN tunnels via speedtest.net

### Supported Platforms

#### On Device

Development was done targeting the Ubiquiti USG and EdgeRouter Lite-3.  These were
picked due to their low cost and relatively limited hardware specs (dual-core
500Mhz MIPS64 CPU and 512MB of RAM).  Hence, you should be able to run
VPNExiter on any
[hardware/OS that GoLang supports](https://github.com/golang/go/wiki/MinimumRequirements).

#### Service

It's easy to run VPNExiter as a service on your NAS or home computer.  VPNExiter
supports controlling the VPN service on your router via SSH.

#### Docker
You can also run VPNExiter on a computer on your network via
[Docker](https://www.docker.com).  This is a more complicated setup, since VPNExiter
has to SSH into your router to reconfigure it, but may be preferable for some
use cases.


### Supported VPNs

Tested VPNs are:

 - StrongSwan
 - OpenVPN

However, any VPN solution that meets the following requirements should be
possible:

 1. There are commands to start, stop and get the status of the service.
 2. There is a command which contains a string that can be used to determine if the VPN is up.
 1. A single file contains the necessary [configuration template information](https://golang.org/pkg/text/template/) to switch
    between VPN exit servers.  Right now, the two pieces of information available to the config file is the `Vendor` name and `VpnServer` which matches the selected IP address or hostname of the selected server.

#### Shamless Plug

If you're looking for a secure VPN provider, I personally like and use
[PersonalVPN from Witopia](https://www.personalvpn.com).  If you're interested
in signing up, you can use my
[referal link]( https://my.personalvpn.com/ref/jfTXEsBs) to get at 15% discount
and I get an equal credit.


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

Configuring VPNExiter is done via a single <em>config.yaml</em> file.  Required fields are in __bold__ and optional fields are in _italics_.

The `listen` block configures how VPNExiter runs

 * __listen:__
    * __http:__ http port
    * _address:_ IP address to listen on.  Default is all interfaces (0.0.0.0)
    * _username:_ http auth username
    * _password:_ http auth password using bcrypt: `htpasswd -nbB <username> <password>`


If you enable `resolve\_servers` for one or more vendors below, set
`dns_refresh_minutes` to a value => 5 to enable refreshing those DNS entries.
 * _dns\_refresh\_minutes:_ minutes between refreshing DNS entries

VPNExiter supports both a browser-based Speedtest URL which can be directly embeded or run the speedtest-cli on the router.

 * _speedtest\_cli:_ path to speedtest cli tool
 * _speedtest\_url:_ path to custom speedtest.net URL.  example: [https://synfin.speedtestcustom.com](https://synfin.speedtestcustom.com)

The `router` block configures how VPNExiter should connect to the router and manage the VPN tunnel.

 * __router:__
    * __mode:__ Should be `ssh` or `local`
    * __config_file:__ Path to VPN config file.  Example: `/etc/ipsec/ipsec.conf`
    * __start_command:__ Command to start VPN service.  Example: `sudo /usr/sbin/ipsec start`
    * __stop_command:__ Command to stop VPN service.  Example: `sudo /usr/sbin/ipsec stop`
    * __status_command:__ Command to query VPN service status.  Example: `/usr/sbin/ipsec status {{.Vendor}}`
    * __check:__
    	* __command:__ Command to query VPN service status.  Example: `/usr/sbin/ipsec status {{.Vendor}}`
    	* __match:__ String to look for.  Example: `CONNECTED`
    * _ssh:_
	   * _host:_ IP address of router to ssh to (default: 192.168.1.1)
	   * _port:_ Port sshd listens on (default 22)
	   * _username:_ ssh username
	   * _password:_ ssh password

The `vendors` block lists all the configured VPN vendors.

 * __vendors:__
    - __*vendor name 1*__
    - _*vendor name 2*_
    - _*vendor name X*_

Each VPN vendor has it's own block.

 * __*vendor name 1*__  // name of vendor.  Must match an item in `vendors`
    * __config\_template:__ path to config template used to configure the VPN tunnel
    * _resolve\_servers:_ `true` | `false` to enable DNS lookup of IP addresses for any hostnames listed as servers.  Default is false.
    * _levels:_ // If your want to group the VPN exits by geography or other manner, you can define the levels here.
        - *level 1*  // example: Region
        - *level 2*  // example: City
    * __servers:__ // list all the servers
        * Level 1: // name of first level.  For example: North America
            * Level 2: // name of second level. For example: San Francisco
                - server 1 // name of the server or IP address
                - server 2
                - server 3
            * Level 2:  // New York
                - server 1
        * Level 1:  // Europe
            * Level 2:  // London
                - server 1
            * Level 2:  // Paris
                - server 1
            * Level 2:  // Berlin
                - server 1
        * Level 1:  // Asia
            * Level 2:  // Tokyo
                - server 1
                - server 2

 * *vendor name 2*
    * config_template: *path to config template*
    * resolve_servers: true|false
    * servers:
        - server 1
        - server 2



