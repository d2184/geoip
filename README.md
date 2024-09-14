# GeoIP 简介

本项目每天早上七点三十自动生成 GeoIP 文件。

## 与官方版 GeoIP 的区别

- 中国 IPv4地址数据使用 [misakaio/chnroutes2](https://github.com/misakaio/chnroutes2)。
- 中国 IPv6地址数据使用 [fernvenue/chn-cidr-list](https://github.com/fernvenue/chn-cidr-list)
- 新增类别（方便有特殊需求的用户使用）：
  - `geoip:cloudflare`
  - `geoip:facebook`
  - `geoip:google`
  - `geoip:netflix`
  - `geoip:telegram`
  - `geoip:twitter`
  - `geoip:tor`

## 注意

由于本项目使用的均为免费数据，在精度上难免存在偏差，如有精准IP定位需求，请自行前往 [ipip.net](https://ipip.net) 等BGP/ASN数据分析服务商购买付费服务。

## License

[CC-BY-SA-4.0](https://creativecommons.org/licenses/by-sa/4.0/) and [GPL-3.0](https://github.com/Loyalsoldier/geoip/blob/master/LICENSE-GPL)

This product includes GeoLite2 data created by MaxMind, available from [MaxMind](http://www.maxmind.com).
