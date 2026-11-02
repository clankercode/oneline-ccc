"""Loopback-only model discovery fixture; records paths, never credentials."""
import argparse
import http.server
import json
import pathlib

parser = argparse.ArgumentParser()
parser.add_argument('--ready-file', required=True)
parser.add_argument('--requests-file', required=True)
args = parser.parse_args()


class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        with open(args.requests_file, 'a', encoding='utf-8') as log:
            log.write(self.path + '\n')
        status = 200
        location = None
        if self.headers.get('Authorization') != 'Bearer fixture-only-not-a-real-credential':
            status, body = 401, b'{}'
        elif self.path == '/ok/models':
            body = json.dumps({'data': [
                {'id': 'fixture-model', 'context_window': 1, 'service_tier': 'fast'},
                {'id': 'fixture-alternative'},
            ]}).encode()
        elif self.path == '/redirect/models':
            status, body, location = 302, b'{}', '/redirect-target'
        elif self.path == '/malformed/models':
            body = b'{not valid json'
        elif self.path == '/missing/models':
            body = b'{"unexpected": true}'
        elif self.path == '/invalid-ids/models':
            body = b'{"data":[{"id":42},{"id":""},{"id":"bad\\nmodel"}]}'
        else:
            status, body = 404, b'{}'
        self.send_response(status)
        if location:
            self.send_header('Location', location)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *_):
        pass


server = http.server.HTTPServer(('127.0.0.1', 0), Handler)
pathlib.Path(args.ready_file).write_text(str(server.server_port), encoding='ascii')
server.serve_forever()
