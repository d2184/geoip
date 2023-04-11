# 简介

本项目每周二、四、六自动生成 GeoIP 文件，同时提供命令行界面（CLI）供用户自行定制 GeoIP 文件，包括但不限于 V2Ray dat 格式路由规则文件 `geoip.dat` 和 MaxMind mmdb 格式文件 `Country.mmdb`。

## 与官方版 GeoIP 的区别

- 中国大陆 IP地址数据使用 [GeoIP2-CN](https://github.com/d2184/GeoIP2-CN)
- 新增类别（方便有特殊需求的用户使用）：
  - `geoip:cloudflare`（`GEOIP,CLOUDFLARE`）
  - `geoip:cloudfront`（`GEOIP,CLOUDFRONT`）
  - `geoip:facebook`（`GEOIP,FACEBOOK`）
  - `geoip:fastly`（`GEOIP,FASTLY`）
  - `geoip:google`（`GEOIP,GOOGLE`）
  - `geoip:netflix`（`GEOIP,NETFLIX`）
  - `geoip:telegram`（`GEOIP,TELEGRAM`）
  - `geoip:twitter`（`GEOIP,TWITTER`）

## 下载地址

### V2Ray dat 格式路由规则文件

> 适用于 [V2Ray](https://github.com/v2fly/v2ray-core)、[Xray-core](https://github.com/XTLS/Xray-core) 和 [Trojan-Go](https://github.com/p4gefau1t/trojan-go)。

- **geoip.dat**：
  - [https://github.com/d2184/geoip/releases/download/geoip/geoip.dat](https://github.com/d2184/geoip/releases/download/geoip/geoip.dat)

- **geoip-only-cn-private.dat**:
  - [https://github.com/d2184/geoip/releases/download/geoip/geoip-only-cn-private.dat](https://github.com/d2184/geoip/releases/download/geoip/geoip-only-cn-private.dat)

### MaxMind mmdb 格式文件

> 适用于 [Clash](https://github.com/Dreamacro/clash) 和 [Leaf](https://github.com/eycorsican/leaf)。

- **Country.mmdb**：
  - [https://github.com/d2184/geoip/releases/download/geoip/Country.mmdb](https://github.com/d2184/geoip/releases/download/geoip/Country.mmdb)

- **Country-only-cn-private.mmdb**:
  - [https://github.com/d2184/geoip/releases/download/geoip/Country-only-cn-private.mmdb](https://github.com/d2184/geoip/releases/download/geoip/Country-only-cn-private.mmdb)

### sing-box 格式文件

> 适用于 [sing-box](https://github.com/SagerNet/sing-box)

- **geoip.db**:
  - [https://github.com/d2184/geoip/raw/release/geoip.db](https://github.com/d2184/geoip/raw/release/geoip.db)
 
- **geoip-cn.db**:
  - [https://github.com/d2184/geoip/raw/release/geoip-cn.db](https://github.com/d2184/geoip/raw/release/geoip-cn.db)

## 参考配置

在 [V2Ray](https://github.com/v2fly/v2ray-core) 中使用本项目 `.dat` 格式文件的参考配置：

```json
"routing": {
  "rules": [
    {
      "type": "field",
      "outboundTag": "Direct",
      "ip": [
        "geoip:cn",
        "geoip:private"
      ]
    },
    {
      "type": "field",
      "outboundTag": "Proxy",
      "ip": [
        "geoip:us",
        "geoip:jp",
        "geoip:facebook",
        "geoip:telegram"
      ]
    }
  ]
}
```

在 [Clash](https://github.com/Dreamacro/clash) 中使用本项目 `.mmdb` 格式文件的参考配置：

```yaml
rules:
  - GEOIP,PRIVATE,policy,no-resolve
  - GEOIP,FACEBOOK,policy
  - GEOIP,CN,policy,no-resolve
```

在[sing-box](https://github.com/SagerNet/sing-box) 中使用本项目 `.db` 格式文件的参考配置:

```json
{
  "route": [
    "rules": [
      {
        "geoip": [
          "telegram",
          "twitter"
        ],
        "outbound": "Proxy"
      },
      {
        "geoip": [
          "cn",
          "private"
        ],
        "outbound": "direct"
      }
    ]
  ]
}
```

在 [Leaf](https://github.com/eycorsican/leaf) 中使用本项目 `.mmdb` 格式文件的参考配置，查看[官方 README](https://github.com/eycorsican/leaf/blob/master/README.zh.md#geoip)。

**特别说明：**

- **在线生成**：如果需要使用 MaxMind GeoLite2 Country CSV 数据文件，需要在自己仓库的 **[Settings]** 选项卡的 **[Secrets]** 页面中添加一个名为 **MAXMIND_GEOLITE2_LICENSE** 的 secret，否则 GitHub Actions 会运行失败。这个 secret 的值为 MAXMIND 账号的 LICENSE KEY，需要[**注册 MAXMIND 账号**](https://www.maxmind.com/en/geolite2/signup)后，在[**个人账号管理页面**](https://www.maxmind.com/en/account)左侧边栏的 **[Services]** 项下的 **[My License Key]** 里生成。
- **本地生成**：如果需要使用 MaxMind GeoLite2 Country CSV 数据文件（`GeoLite2-Country-CSV.zip`），需要提前从 MaxMind 下载，或从本项目 [release 分支](https://github.com/Loyalsoldier/geoip/tree/release)[下载](https://github.com/Loyalsoldier/geoip/raw/release/GeoLite2-Country-CSV.zip)，并解压缩到名为 `geolite2` 的目录。

### 概念解析

本项目有两个概念：`input` 和 `output`。`input` 指数据源（data source）及其输入格式，`output` 指数据的去向（data destination）及其输出格式。CLI 的作用就是通过读取配置文件中的选项，聚合用户提供的所有数据源，去重，将其转换为目标格式，并输出到文件。

### 支持的格式

关于每种格式所支持的配置选项，查看本项目 [`config-example.json`](https://github.com/Loyalsoldier/geoip/blob/HEAD/config-example.json) 文件。

支持的 `input` 输入格式：

- **text**：纯文本 IP 和 CIDR（例如：`1.1.1.1` 或 `1.0.0.0/24`）
- **private**：局域网和私有网络 CIDR（例如：`192.168.0.0/16` 和 `127.0.0.0/8`）
- **cutter**：用于裁剪前置步骤中的数据
- **v2rayGeoIPDat**：V2Ray GeoIP dat 格式（`geoip.dat`）
- **maxmindMMDB**：MaxMind mmdb 数据格式（`GeoLite2-Country.mmdb`）
- **maxmindGeoLite2CountryCSV**：MaxMind GeoLite2 country CSV 数据（`GeoLite2-Country-CSV.zip`）
- **clashRuleSetClassical**：[classical 类型的 Clash RuleSet](https://github.com/Dreamacro/clash/wiki/premium-core-features#classical)
- **clashRuleSet**：[ipcidr 类型的 Clash RuleSet](https://github.com/Dreamacro/clash/wiki/premium-core-features#ipcidr)
- **surgeRuleSet**：[Surge RuleSet](https://manual.nssurge.com/rule/ruleset.html)

支持的 `output` 输出格式：

- **text**：纯文本 CIDR（例如：`1.0.0.0/24`）
- **v2rayGeoIPDat**：V2Ray GeoIP dat 格式（`geoip.dat`，适用于 [V2Ray](https://github.com/v2fly/v2ray-core)、[Xray-core](https://github.com/XTLS/Xray-core) 和 [Trojan-Go](https://github.com/p4gefau1t/trojan-go)）
- **maxmindMMDB**：MaxMind mmdb 数据格式（`GeoLite2-Country.mmdb`，适用于 [Clash](https://github.com/Dreamacro/clash) 和 [Leaf](https://github.com/eycorsican/leaf)）
- **clashRuleSetClassical**：[classical 类型的 Clash RuleSet](https://github.com/Dreamacro/clash/wiki/premium-core-features#classical)
- **clashRuleSet**：[ipcidr 类型的 Clash RuleSet](https://github.com/Dreamacro/clash/wiki/premium-core-features#ipcidr)
- **surgeRuleSet**：[Surge RuleSet](https://manual.nssurge.com/rule/ruleset.html)

### 注意事项

由于 MaxMind mmdb 文件格式的限制，当不同列表的 IP 或 CIDR 数据有交集或重复项时，后写入的列表的 IP 或 CIDR 数据会覆盖（overwrite）之前已写入的列表的数据。譬如，IP `1.1.1.1` 同属于列表 `AU` 和列表 `Cloudflare`。如果 `Cloudflare` 在 `AU` 之后写入，则 IP `1.1.1.1` 归属于列表 `Cloudflare`。

为了确保某些指定的列表、被修改的列表一定囊括属于它的所有 IP 或 CIDR 数据，可在 `output` 输出格式为 `maxmindMMDB` 的配置中增加选项 `overwriteList`，该选项中指定的列表会在最后逐一写入，列表中最后一项优先级最高。若已设置选项 `wantedList`，则无需设置 `overwriteList`。`wantedList` 中指定的列表会在最后逐一写入，列表中最后一项优先级最高。

## CLI 功能展示

可通过 `go install -v github.com/d2184/geoip@latest` 直接安装 CLI。

```bash
$ ./geoip -h
Usage of ./geoip:
  -c string
    	URI of the JSON format config file, support both local file path and remote HTTP(S) URL (default "config.json")
  -l	List all available input and output formats

$ ./geoip -c config.json
2021/08/29 12:11:35 ✅ [v2rayGeoIPDat] geoip.dat --> output/dat
2021/08/29 12:11:35 ✅ [v2rayGeoIPDat] geoip-only-cn-private.dat --> output/dat
2021/08/29 12:11:35 ✅ [v2rayGeoIPDat] geoip-asn.dat --> output/dat
2021/08/29 12:11:35 ✅ [v2rayGeoIPDat] cn.dat --> output/dat
2021/08/29 12:11:35 ✅ [v2rayGeoIPDat] private.dat --> output/dat
2021/08/29 12:11:39 ✅ [maxmindMMDB] Country.mmdb --> output/maxmind
2021/08/29 12:11:39 ✅ [maxmindMMDB] Country-only-cn-private.mmdb --> output/maxmind
2021/08/29 12:11:39 ✅ [text] netflix.txt --> output/text
2021/08/29 12:11:39 ✅ [text] telegram.txt --> output/text
2021/08/29 12:11:39 ✅ [text] cn.txt --> output/text
2021/08/29 12:11:39 ✅ [text] cloudflare.txt --> output/text
2021/08/29 12:11:39 ✅ [text] cloudfront.txt --> output/text
2021/08/29 12:11:39 ✅ [text] facebook.txt --> output/text
2021/08/29 12:11:39 ✅ [text] fastly.txt --> output/text

$ ./geoip -l
All available input formats:
  - v2rayGeoIPDat (Convert V2Ray GeoIP dat to other formats)
  - maxmindMMDB (Convert MaxMind mmdb database to other formats)
  - maxmindGeoLite2CountryCSV (Convert MaxMind GeoLite2 country CSV data to other formats)
  - private (Convert LAN and private network CIDR to other formats)
  - text (Convert plaintext IP & CIDR to other formats)
  - clashRuleSetClassical (Convert classical type of Clash RuleSet to other formats (just processing IP & CIDR lines))
  - clashRuleSet (Convert ipcidr type of Clash RuleSet to other formats)
  - surgeRuleSet (Convert Surge RuleSet to other formats (just processing IP & CIDR lines))
  - cutter (Remove data from previous steps)
  - test (Convert specific CIDR to other formats (for test only))
All available output formats:
  - v2rayGeoIPDat (Convert data to V2Ray GeoIP dat format)
  - maxmindMMDB (Convert data to MaxMind mmdb database format)
  - clashRuleSetClassical (Convert data to classical type of Clash RuleSet)
  - clashRuleSet (Convert data to ipcidr type of Clash RuleSet)
  - surgeRuleSet (Convert data to Surge RuleSet)
  - text (Convert data to plaintext CIDR format)
```

## License

[CC-BY-SA-4.0](https://creativecommons.org/licenses/by-sa/4.0/)

This product includes GeoLite2 data created by MaxMind, available from [MaxMind](http://www.maxmind.com).
