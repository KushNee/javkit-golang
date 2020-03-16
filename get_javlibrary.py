import argparse
from time import sleep
from typing import List, Union

import cloudscraper
from requests import Response


def get_jav_html(url_list: List[Union[int, str]]) -> str:
    """获取 javlibrary 网页内容；使用 cloudscraper 跳过 cloudflare 验证

    :param url_list:[0]-errorTimes,[1]-url,[2]-proxy
    :return: scraper.text
    """
    scraper = cloudscraper.create_scraper(browser="chrome")
    while url_list[0] != 6:
        try:
            rqs: Response = Response()
            if len(url_list) == 2:
                rqs = scraper.get(url_list[1])
            elif len(url_list) == 3:
                rqs = scraper.get(url_list[1], proxies=url_list[2])
            rqs.encoding = 'utf-8'
            return rqs.text
        except Exception as e:
            sleep(5)
            if url_list[0] == 5:
                raise e
            url_list[0] += 1


parser=argparse.ArgumentParser()
parser.add_argument("--url",type=str,default=None)
args=parser.parse_args()
url_list=[0,args.url]
try:
    src=get_jav_html(url_list)
    print(src)
except Exception as e:
    print(str(e))

