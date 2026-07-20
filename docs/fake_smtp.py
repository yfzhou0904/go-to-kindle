"""One-shot authenticated TLS SMTP sink used by the README demo."""

import socket
import ssl
import sys


def reply(connection, message):
    connection.sendall((message + "\r\n").encode())


context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
context.load_cert_chain(sys.argv[1], sys.argv[2])

with socket.create_server(("127.0.0.1", 2465)) as server:
    with context.wrap_socket(server.accept()[0], server_side=True) as connection:
        reply(connection, "220 localhost demo SMTP")
        data_mode = False
        stream = connection.makefile("rb")
        while line := stream.readline():
            command = line.decode(errors="replace").rstrip("\r\n")
            if data_mode:
                if command == ".":
                    data_mode = False
                    reply(connection, "250 message accepted")
                continue
            verb = command.split(" ", 1)[0].upper()
            if verb in {"EHLO", "HELO"}:
                reply(connection, "250-localhost")
                reply(connection, "250 AUTH PLAIN")
            elif verb == "AUTH":
                reply(connection, "235 authentication successful")
            elif verb in {"MAIL", "RCPT"}:
                reply(connection, "250 OK")
            elif verb == "DATA":
                data_mode = True
                reply(connection, "354 end with <CRLF>.<CRLF>")
            elif verb == "QUIT":
                reply(connection, "221 goodbye")
                break
            else:
                reply(connection, "250 OK")
