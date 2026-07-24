#include "vps_pilot/linux_metrics.hpp"

#include <sys/statvfs.h>
#include <sys/sysinfo.h>
#include <sys/utsname.h>

#include <algorithm>
#include <cctype>
#include <chrono>
#include <fstream>
#include <sstream>
#include <string>
#include <thread>

namespace vps_pilot {
namespace {

std::string unquote(std::string value) {
    if (value.size() >= 2 && value.front() == '"' && value.back() == '"') {
        return value.substr(1, value.size() - 2);
    }
    return value;
}

std::vector<LinuxMetrics::CpuTimes> read_cpu_times() {
    std::ifstream input("/proc/stat");
    std::vector<LinuxMetrics::CpuTimes> times;
    std::string line;

    while (std::getline(input, line)) {
        if (!line.starts_with("cpu")) {
            break;
        }
        std::istringstream stream(line);
        std::string name;
        stream >> name;
        if (name == "cpu") {
            continue;
        }

        std::uint64_t user{}, nice{}, system{}, idle{}, iowait{}, irq{}, softirq{}, steal{};
        stream >> user >> nice >> system >> idle >> iowait >> irq >> softirq >> steal;
        const auto busy = user + nice + system + irq + softirq + steal;
        times.push_back({busy, busy + idle + iowait});
    }
    return times;
}

std::pair<std::uint64_t, std::uint64_t> read_network_bytes() {
    std::ifstream input("/proc/net/dev");
    std::string line;
    std::uint64_t received{};
    std::uint64_t sent{};

    while (std::getline(input, line)) {
        const auto colon = line.find(':');
        if (colon == std::string::npos) {
            continue;
        }

        auto interface_name = line.substr(0, colon);
        interface_name.erase(
            std::remove_if(interface_name.begin(), interface_name.end(), [](unsigned char value) {
                return std::isspace(value) != 0;
            }),
            interface_name.end());
        if (interface_name == "lo") {
            continue;
        }

        std::istringstream stream(line.substr(colon + 1));
        std::uint64_t recv_bytes{};
        std::uint64_t field{};
        std::uint64_t sent_bytes{};
        stream >> recv_bytes;
        for (int index = 0; index < 7; ++index) {
            stream >> field;
        }
        stream >> sent_bytes;
        received += recv_bytes;
        sent += sent_bytes;
    }
    return {sent, received};
}

}  // namespace

nlohmann::json LinuxMetrics::system_info() const {
    std::string platform{"linux"};
    std::string platform_version;
    std::ifstream release("/etc/os-release");
    std::string line;
    while (std::getline(release, line)) {
        const auto equals = line.find('=');
        if (equals == std::string::npos) {
            continue;
        }
        const auto key = line.substr(0, equals);
        const auto value = unquote(line.substr(equals + 1));
        if (key == "ID") {
            platform = value;
        } else if (key == "VERSION_ID") {
            platform_version = value;
        }
    }

    utsname kernel{};
    const auto kernel_version = uname(&kernel) == 0 ? std::string(kernel.release) : "unknown";

    struct sysinfo memory {};
    const auto total_memory = ::sysinfo(&memory) == 0
                                  ? static_cast<std::uint64_t>(memory.totalram) * memory.mem_unit
                                  : 0;

    return {
        {"os", "linux"},
        {"platform", platform},
        {"platform_version", platform_version},
        {"kernel_version", kernel_version},
        {"cpus", std::thread::hardware_concurrency()},
        {"total_memory", total_memory},
    };
}

nlohmann::json LinuxMetrics::sample() {
    const auto current_cpu = read_cpu_times();
    std::vector<double> cpu_usage(current_cpu.size(), 0.0);
    if (previous_cpu_.size() == current_cpu.size()) {
        for (std::size_t index = 0; index < current_cpu.size(); ++index) {
            const auto total_delta = current_cpu[index].total - previous_cpu_[index].total;
            const auto busy_delta = current_cpu[index].busy - previous_cpu_[index].busy;
            if (total_delta > 0) {
                cpu_usage[index] =
                    100.0 * static_cast<double>(busy_delta) / static_cast<double>(total_delta);
            }
        }
    }
    previous_cpu_ = current_cpu;

    struct sysinfo memory {};
    double memory_usage{};
    if (::sysinfo(&memory) == 0 && memory.totalram > 0) {
        memory_usage =
            100.0 * static_cast<double>(memory.totalram - memory.freeram - memory.bufferram) /
            static_cast<double>(memory.totalram);
    }

    statvfs disk{};
    double disk_usage{};
    if (statvfs("/", &disk) == 0 && disk.f_blocks > 0) {
        disk_usage = 100.0 * static_cast<double>(disk.f_blocks - disk.f_bavail) /
                     static_cast<double>(disk.f_blocks);
    }

    const auto [sent, received] = read_network_bytes();
    const auto now = std::chrono::steady_clock::now();
    std::int64_t sent_per_second{};
    std::int64_t received_per_second{};
    if (has_previous_net_) {
        const auto elapsed = std::chrono::duration<double>(now - previous_net_time_).count();
        if (elapsed > 0.0) {
            sent_per_second = static_cast<std::int64_t>(
                static_cast<double>(sent - previous_net_sent_) / elapsed);
            received_per_second = static_cast<std::int64_t>(
                static_cast<double>(received - previous_net_received_) / elapsed);
        }
    }
    previous_net_sent_ = sent;
    previous_net_received_ = received;
    previous_net_time_ = now;
    has_previous_net_ = true;

    return {
        {"cpu_usage", cpu_usage},
        {"mem_usage", memory_usage},
        {"disk_usage", disk_usage},
        {"net_sent_ps", sent_per_second},
        {"net_recv_ps", received_per_second},
    };
}

}  // namespace vps_pilot
