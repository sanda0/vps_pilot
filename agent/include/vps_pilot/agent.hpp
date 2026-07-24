#pragma once

#include "vps_pilot/config.hpp"
#include "vps_pilot/linux_metrics.hpp"
#include "vps_pilot/project_scanner.hpp"

#include <asio.hpp>
#include <cstdint>
#include <nlohmann/json.hpp>

namespace vps_pilot {

class Agent {
public:
    explicit Agent(Config config);
    void run();

private:
    void connect();
    void session();
    void send_message(const std::string& type, const nlohmann::json& data);
    nlohmann::json receive_message();

    Config config_;
    asio::io_context io_;
    asio::ip::tcp::socket socket_{io_};
    LinuxMetrics metrics_;
    ProjectScanner projects_;
    std::int32_t node_id_{};
};

}  // namespace vps_pilot

