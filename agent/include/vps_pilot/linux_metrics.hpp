#pragma once

#include <cstdint>
#include <chrono>
#include <vector>

#include <nlohmann/json.hpp>

namespace vps_pilot {

class LinuxMetrics {
public:
    struct CpuTimes {
        std::uint64_t busy{};
        std::uint64_t total{};
    };

    nlohmann::json system_info() const;
    nlohmann::json sample();

private:
    std::vector<CpuTimes> previous_cpu_;
    std::uint64_t previous_net_sent_{};
    std::uint64_t previous_net_received_{};
    bool has_previous_net_{false};
    std::chrono::steady_clock::time_point previous_net_time_{};
};

}  // namespace vps_pilot
