Building a legacy search engine for a legacy protocol
===

Most users who use the internet today, are mostly focused on three protocols, HTTP, TLS, and DNS.

While these users may not care these days how their pages are displayed to them, there was once a competing protocol to the protocol that we known and love that is HTTP.

Location just 10 ports down from HTTP, is [Gopher](https://www.ietf.org/rfc/rfc1436.txt) on TCP port 70, A protocol that looks a lot like a much more basic HTTP/1.0 while a basic HTTP/1.0 request may look like this:

```

$ nc blog.benjojo.co.uk 80
GET / HTTP/1.0


HTTP/1.1 403 Forbidden
Date: Wed, 03 May 2017 19:11:28 GMT
Content-Type: text/html; charset=UTF-8
Connection: close

```

A gopher request is even more basic:

```
$ echo "/archive" | nc gopher.floodgap.com 70
1Floodgap Systems gopher root	/	gopher.floodgap.com	70
i 		error.host	1
iWelcome to the Floodgap gopher files archive. This contains		error.host	1
imirrors of popular or departed archives that we believe useful		error.host	1
```

This is great for basic file transfers, as basic bash utilities can be used to download files

However in the end, HTTP won out over gopher and it became the protocol that most of us use to do things on the internet. The reasons why are interesting, however others have told that story much better, [you can find a good write up here](https://www.minnpost.com/business/2016/08/rise-and-fall-gopher-protocol)

Giving that search engines exist for HTTP, and gopher itself has support for searching in the protocol, I realised that while there are search engines for gopher, in "gopherspace" there was not one that really existed in HTTP (that I could find)

Starting that, I had a friend run a [massscan](https://github.com/robertdavidgraham/masscan) over the internet for port 70, and then filtered the results for real gopher servers:

```
[ben@aura tmp]$ pv gopher.raw | grep 'read":"i' | jq .ip | sort -n | uniq -c | wc -l
2.14GiB 0:00:08 [ 254MiB/s] [================>] 100%            
370
```

A sad 370 servers are left on the internet that serve gopher.

## Building a crawler

A simple rfc1436 implementation was written and a crawler began (slowly, there are very old servers behind some of these hosts) crawling all the menus (known as selectors) and text files I could find.

At this point I started to explore gopher space itself, and I have to say, It's a wonderful place of just pure content, a far cry away from the modern internet where CSS and adtech is stuffed in every corner

![A ttygif of using gopher](link/to/ttygif.gif)

## Indexing the content

Giving that gopher is from the 1990's, it feels only right to use search engine tech from the era, as it happens [AltaVista](https://en.wikipedia.org/wiki/AltaVista) once sold a personal/home version of their search engine for 
desktop computers. The issue however it is win32 only software, I didn't try running it on wine, instead I vouched for a more authentic experience of running the software:

Using a already [fantastic guide from NeoZeed](https://virtuallyfun.superglobalmegacorp.com/2017/02/25/personal-altavista-utzoo-reloaded/) I ended up provisioning my Windows 98 search "server"

![altavista being installed]()

Confirming that the search engine works:

![altavista test search]()

As NeoZeed found out, the search interface only listens on loopback (hardcoded), this is very annoying if you want to expose it to the wider world! To solve this [stunnel](https://www.stunnel.org/index.html) 
was deployed to listen on * and relay connections back to the local instance.

## Sanitise the index 


## Provide data to the indexer

## Final flow

```
                                    Internet

                                        +
                                        |
                                        |
                                        |
                                        |
                                        v

+-----------------+     +---------------------------------+
|                 |     |                                 |
|  alta_sanitise  | <---+         nginx/lighttpd          |
|                 |     |                                 |
+--------+--------+     +---------------------------------+
         |
         |
         |
+--------+------------------------------------------------+
|        |                                                |
|        v   QEMU ( with userspace net and port fwd )     |
|                                                         |
|   +-----------+    +--------------------------------+   |
|   |           |    |                                |   |
|   |           |    |                                |   |
|   |           |    |                                |   |
|   |           |    |                                |   |
|   |           |    |                                |   |
|   |  stunnel  +--> |      AltaVista interface       |   |
|   |           |    |                                |   |
|   |           |    |                                |   |
|   |           |    |                                |   |
|   |           |    |                                |   |
|   |           |    |                                |   |
|   |           |    |                                |   |
|   +-----------+    +--------------------------------+   |
|                                                         |
|                                                         |
+---------------------------------------------------------+
```