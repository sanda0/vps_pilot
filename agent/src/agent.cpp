#include "vps_pilot/agent.hpp"

#include <array>
#include <chrono>
#include <iostream>
#include <limits>
#include <stdexcept>
#include <thread>

namespace vps_pilot {
namespace {

constexpr std::uint32_t max_frame_size = 4U * 1024U * 1024U;

std::array<unsigned char, 4> encode_size(std::uint32_t size) {
    return {
        static_cast<unsigned char>((size >> 24U) & 0xffU),
        static_cast<unsigned char>((size >> 16U) & 0xffU),
        static_cast<unsigned char>((size >> 8U) & 0xffU),
        static_cast<unsigned char>(size & 0xffU),
    };
}

std::uint32_t decode_size(const std::array<unsigned char, 4>& bytes) {
    return (static_cast<std::uint32_t>(bytes[0]) << 24U) |
           (static_cast<std::uint32_t>(bytes[1]) << 16U) |
           (static_cast<std::uint32_t>(bytes[2]) << 8U) |
           static_cast<std::uint32_t>(bytes[3]);
}

}  // namespace

Agent::Agent(Config config)
    : config_(std::move(config)), projects_(config_.project_roots) {}

void Agent::run() {
    std::chrono::seconds retry_delay{1};
    while (true) {
        try {
            connect();
            retry_delay = std::chrono::seconds{1};
            session();
        } catch (const std::exception& exception) {
            std::cerr << "agent connection error: " << exception.what() << "; retrying in "
                      << retry_delay.count() << "s\n";
            asio::error_code ignored;
            socket_.close(ignored);
            std::this_thread::sleep_for(retry_delay);
            retry_delay = std::min(retry_delay * 2, std::chrono::seconds{30});
        }
    }
}

void Agent::connect() {
    node_id_ = 0;
    asio::ip::tcp::resolver resolver(io_);
    const auto endpoints =
        resolver.resolve(config_.server_host, std::to_string(config_.server_port));
    asio::connect(socket_, endpoints);
    socket_.set_option(asio::ip::tcp::no_delay(true));
    std::cout << "connected to " << config_.server_host << ':' << config_.server_port << '\n';
}

void Agent::session() {
    send_message("connected", metrics_.system_info());
    const auto response = receive_message();
    if (response.value("type", "") != "sys_stat") {
        throw std::runtime_error("server did not acknowledge agent handshake");
    }
    node_id_ = response.value("node_id", 0);
    if (node_id_ <= 0) {
        throw std::runtime_error("server returned an invalid node ID");
    }
    std::cout << "registered as node " << node_id_ << '\n';

    send_message("projects", projects_.scan());
    auto next_metrics = std::chrono::steady_clock::now();
    auto next_projects =
        next_metrics + std::chrono::seconds(config_.projects_interval_seconds);

    while (true) {
        const auto now = std::chrono::steady_clock::now();
        if (now >= next_metrics) {
            send_message("sys_stat", metrics_.sample());
            next_metrics = now + std::chrono::seconds(config_.metrics_interval_seconds);
        }
        if (now >= next_projects) {
            send_message("projects", projects_.scan());
            next_projects = now + std::chrono::seconds(config_.projects_interval_seconds);
        }
        std::this_thread::sleep_for(std::chrono::milliseconds{100});
    }
}

void Agent::send_message(const std::string& type, const nlohmann::json& data) {
    nlohmann::json message{{"type", type}, {"data", data}};
    if (!config_.agent_token.empty()) {
        message["token"] = config_.agent_token;
    }
    if (node_id_ > 0) {
        message["node_id"] = node_id_;
    }
    const auto payload = message.dump();
    if (payload.empty() || payload.size() > max_frame_size ||
        payload.size() > std::numeric_limits<std::uint32_t>::max()) {
        throw std::runtime_error("outgoing message exceeds protocol limit");
    }

    const auto header = encode_size(static_cast<std::uint32_t>(payload.size()));
    asio::write(socket_, asio::buffer(header));
    asio::write(socket_, asio::buffer(payload));
}

nlohmann::json Agent::receive_message() {
    std::array<unsigned char, 4> header{};
    asio::read(socket_, asio::buffer(header));
    const auto size = decode_size(header);
    if (size == 0 || size > max_frame_size) {
        throw std::runtime_error("server sent an invalid frame size");
    }

    std::string payload(size, '\0');
    asio::read(socket_, asio::buffer(payload));
    return nlohmann::json::parse(payload);
}

}  // namespace vps_pilot
