DIR=peerswaprpc

PROTOC_OPTS= \
		-I. \
		--go_out=paths=source_relative:. \
		--go-grpc_out=paths=source_relative:. \
		--grpc-gateway_out=. \
    	--grpc-gateway_opt logtostderr=true \
    	--grpc-gateway_opt paths=source_relative \
    	--grpc-gateway_opt generate_unbound_methods=true \
		--grpc-gateway_opt grpc_api_configuration=${DIR}/peerswap.yaml \
		--openapiv2_out=. \
    	--openapiv2_opt logtostderr=true

STUBS= \
		${DIR}/peerswaprpc_grpc.pb.go \
		${DIR}/peerswaprpc.pb.go \
		${DIR}/peerswaprpc.pb.gw.go \
		${DIR}/peerswaprpc.swagger.json

all-rpc: ${STUBS}

clean-rpc:
		rm -f ${STUBS}

${DIR}/peerswaprpc_grpc.pb.go ${DIR}/peerswaprpc.pb.go ${DIR}/peerswaprpc.pb.gw.go ${DIR}/peerswaprpc.swagger.json: ${DIR}/peerswaprpc.proto
	protoc ${PROTOC_OPTS} ${DIR}/peerswaprpc.proto
PHONY: