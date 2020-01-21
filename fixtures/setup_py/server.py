import http.server
import funniest

class SimpleRequestHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        print(self.path)
        self.wfile.write(('HTTP/1.1 200 Okay\r\n\r\n'+funniest.joke()).encode(encoding='utf_8'))

def run(server_class=http.server.HTTPServer,
    handler_class=SimpleRequestHandler):
    server_address = ('', 8080)
    httpd = server_class(server_address, handler_class)
    httpd.serve_forever()

run()
