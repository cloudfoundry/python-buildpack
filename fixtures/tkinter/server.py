import BaseHTTPServer
import Tkinter

class SimpleRequestHandler(BaseHTTPServer.BaseHTTPRequestHandler):
    def do_GET(self):
        print self.path
        self.wfile.write('HTTP/1.1 200 Okay\r\n\r\ntkinter was imported\r\n')

def run(server_class=BaseHTTPServer.HTTPServer,
    handler_class=SimpleRequestHandler):
    server_address = ('', 8080)
    httpd = server_class(server_address, handler_class)
    httpd.serve_forever()

run()
