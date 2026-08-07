//go:build ignore
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>

char __license[] SEC("license") = "Dual MIT/GPL";

// example: https://github.com/xdp-project/xdp-tutorial/blob/main/advanced03-AF_XDP/af_xdp_kern.c

// XSKMAP: special map type that holds AF_XDP socket file descriptors,
// indexed by RX queue id. The XDP program looks up the queue id it's
// currently processing and, if a socket is registered for that queue,
// redirects the packet straight to userspace via that socket.
struct {
    __uint(type, BPF_MAP_TYPE_XSKMAP);
    __uint(max_entries, 64); // supports up to 64 RX queues
    __type(key, __u32);      // queue id
    __type(value, __u32);    // socket fd
} xsks_map SEC(".maps"); // will be objs.XsksMap

SEC("xdp")
int xdp_redirect_to_socket(struct xdp_md *ctx) {
    // ctx->rx_queue_index tells us which hardware queue this packet
    // arrived on. Since AF_XDP sockets bind to a specific queue, this
    // is how the kernel knows which socket (if any) should receive it.
    __u32 index = ctx->rx_queue_index;

    // bpf_map_lookup_elem returns NULL if no socket is registered for
    // this queue id yet in that case we just let the packet continue
    // through the normal network stack instead of dropping it.
    if (bpf_map_lookup_elem(&xsks_map, &index)) {
        // bpf_redirect_map hands the packet to the AF_XDP socket bound
        // to this queue. The kernel copies (or zero-copies, if the
        // driver supports it) the frame into that socket's UMEM.
        return bpf_redirect_map(&xsks_map, index, 0);
    }

    return XDP_PASS;
}