//go:build ignore
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>
#define IPPROTO_UDP 17

// guide: https://ebpf-go.dev/guides/getting-started/#ebpf-c-program

char __license[] SEC("license") = "Dual MIT/GPL";

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 65536);
    __type(key, __u16);   // porta de destino
    __type(value, __u64); // contagem
} port_count SEC(".maps");

// about the reapeated bound checker:
//  we need to prove to the kernel verifier, that matematically
//  we will never read from a memory that doesnt exists

SEC("xdp")
int count_by_port(struct xdp_md *ctx) {
    // debug in development only, dont use it in production, this is costly
    // it writes on a special trace pipe on the kernel
    bpf_printk("packet received\n");

    // ctx-> data and ctx->data_end are pointers from start to finish of the package
    // in memmory. The kernel gives access as "long" (number), we them convert
    // it to void* to do arithmetic pointer manipulation
    void *data = (void *)(long) ctx->data;
    void *data_end = (void *)(long) ctx->data_end;

    // reinterpreting the first bytes of the package like a Ethernet header
    // (this doesnt copy anything, it just "reads the bytes on this layout")
    struct ethhdr *eth = data;

    // needed bounds check: "eth + 1" is a pointer to the byte after
    // the end of struct ethhdr (this is pointer arithmetic in C)
    // if it passes data_end, the package is smaller them the entire Ethernet header
    // so we cant safely read it, just give up and let it pass
    if ((void *)(eth + 1) > data_end)
        return XDP_PASS;

    // Ethernet header always have a fixed size, so IP header starts
    // just right after him ("eth + 1"), here is used as address, not as
    // comparison: its the "pointer just after the end of eth"
    struct iphdr *ip = (void *)(eth + 1);
    // same bounds check, now for IP header
    if ((void *)(ip + 1) > data_end)
        return XDP_PASS;

    // we only want UDP packets, so ip->protocol says which is the next protocol
    // after IP (TCP, UDP, ICMP...)
    if (ip->protocol != IPPROTO_UDP)
        return XDP_PASS;

    // IMPORTANT: IP header can have a varying size (because of field "options" from each protocol)
    // so ip->ihl sayus the real size of it, in unities of 4 bytes, thats why we multiply by 4
    // we never assumes sizeof(*ip) fixed, because this ignores the options when they exist
    struct udphdr *udp = (void *)ip + (ip -> ihl * 4);
    // bound check again, but for the UDP header
    if ((void *)(udp + 1) > data_end)
        return XDP_PASS;

    // udp-> dest comes as "network byte order" (big-endian as industry standard)
    // but the machine CPU can be little-endian, bpf_ntohs converts to the desired format
    // formats that only makes sense if you compare/use like a real number
    __u16 port = bpf_ntohs(udp->dest);

    // we search the eBPF map (we share with the Go in userspace) if it already exists
    // on an counting entry for this port. It returns the POINTER to the value inside the map,
    // or NULL if the key doesnt exists yet
    __u64 *count = bpf_map_lookup_elem(&port_count, &port);
    if (count) {
        // the counting for this port already exists, we increment atomatically
        // "atomatically" imports because the packages can arrive in parallel
        // (múltiple queues of network/CPUs), without it we could lost the counting
        // because of the race condition between concurrent increment
        __sync_fetch_and_add(count, 1);
    } else {
        // first time on this port, we create a new map entry with initial value as 1
        // BPF_ANY = creates if it doesnt exist, update if it does
        __u64 initial = 1;
        bpf_map_update_elem(&port_count, &port, &initial, BPF_ANY);
    }

    // XDP_PASS = "processed what was needed, let the package follors normally to the rest of
    // the network stack on the kernel".
    // On this code is just couniting, we dont filter, and doesnt drop nothing yet
    return XDP_PASS;
}