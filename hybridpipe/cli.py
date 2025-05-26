#!/usr/bin/env python3
"""
Command-line interface for HybridPipe.

This module provides a CLI for testing, monitoring, and managing
HybridPipe messaging operations.
"""

import asyncio
import json
import sys
from typing import Any, Dict, Optional
from datetime import datetime

import click
from rich.console import Console
from rich.table import Table
from rich.json import JSON
from rich.live import Live
from rich.panel import Panel

from hybridpipe import (
    deploy_router,
    BrokerType,
    get_supported_protocols,
    get_metrics_collector,
    MessageMetadata,
)

console = Console()


@click.group()
@click.version_option(version="2.0.0")
def cli():
    """HybridPipe - Unified messaging interface for microservices."""
    pass


@cli.command()
def list_protocols():
    """List all supported messaging protocols."""
    protocols = get_supported_protocols()
    
    table = Table(title="Supported Protocols")
    table.add_column("Protocol", style="cyan")
    table.add_column("ID", style="magenta")
    table.add_column("Status", style="green")
    
    for protocol in protocols:
        table.add_row(
            protocol.name,
            str(protocol.value),
            "Available"
        )
    
    console.print(table)


@cli.command()
@click.argument("protocol", type=click.Choice([p.name.lower() for p in BrokerType]))
@click.option("--config", "-c", help="Configuration JSON string or file path")
@click.option("--pipe", "-p", default="test.channel", help="Pipe/channel name")
@click.option("--message", "-m", default='{"test": "message"}', help="Message to send (JSON)")
@click.option("--count", "-n", default=1, help="Number of messages to send")
def send(protocol: str, config: Optional[str], pipe: str, message: str, count: int):
    """Send messages using the specified protocol."""
    
    async def send_messages():
        # Parse protocol
        broker_type = BrokerType[protocol.upper()]
        
        # Parse config
        config_dict = {}
        if config:
            try:
                if config.startswith("{"):
                    config_dict = json.loads(config)
                else:
                    with open(config, 'r') as f:
                        config_dict = json.load(f)
            except Exception as e:
                console.print(f"[red]Error parsing config: {e}[/red]")
                return
        
        # Parse message
        try:
            message_data = json.loads(message)
        except json.JSONDecodeError as e:
            console.print(f"[red]Error parsing message JSON: {e}[/red]")
            return
        
        # Deploy router
        try:
            router = await deploy_router(broker_type, config=config_dict)
            console.print(f"[green]Connected to {protocol.upper()} broker[/green]")
        except Exception as e:
            console.print(f"[red]Failed to connect: {e}[/red]")
            return
        
        try:
            # Send messages
            with console.status(f"Sending {count} messages..."):
                start_time = datetime.now()
                
                for i in range(count):
                    msg = message_data.copy()
                    if count > 1:
                        msg["sequence"] = i
                        msg["timestamp"] = datetime.now().isoformat()
                    
                    await router.dispatch(pipe, msg)
                
                end_time = datetime.now()
                duration = (end_time - start_time).total_seconds()
            
            console.print(f"[green]Sent {count} messages in {duration:.3f} seconds[/green]")
            console.print(f"[cyan]Rate: {count / duration:.1f} messages/second[/cyan]")
            
        finally:
            await router.disconnect()
    
    asyncio.run(send_messages())


@cli.command()
@click.argument("protocol", type=click.Choice([p.name.lower() for p in BrokerType]))
@click.option("--config", "-c", help="Configuration JSON string or file path")
@click.option("--pipe", "-p", default="test.channel", help="Pipe/channel name")
@click.option("--timeout", "-t", default=30, help="Timeout in seconds")
def listen(protocol: str, config: Optional[str], pipe: str, timeout: int):
    """Listen for messages using the specified protocol."""
    
    async def listen_messages():
        # Parse protocol
        broker_type = BrokerType[protocol.upper()]
        
        # Parse config
        config_dict = {}
        if config:
            try:
                if config.startswith("{"):
                    config_dict = json.loads(config)
                else:
                    with open(config, 'r') as f:
                        config_dict = json.load(f)
            except Exception as e:
                console.print(f"[red]Error parsing config: {e}[/red]")
                return
        
        # Deploy router
        try:
            router = await deploy_router(broker_type, config=config_dict)
            console.print(f"[green]Connected to {protocol.upper()} broker[/green]")
        except Exception as e:
            console.print(f"[red]Failed to connect: {e}[/red]")
            return
        
        received_count = 0
        
        async def message_handler(data: bytes, metadata: MessageMetadata):
            nonlocal received_count
            received_count += 1
            
            from hybridpipe.serialization.engine import decode
            message = decode(data)
            
            console.print(Panel(
                JSON.from_data(message),
                title=f"Message {received_count} - {metadata.message_id[:8]}",
                subtitle=f"Received at {metadata.timestamp.strftime('%H:%M:%S')}"
            ))
        
        try:
            # Subscribe to channel
            await router.subscribe(pipe, message_handler)
            console.print(f"[cyan]Listening on '{pipe}' for {timeout} seconds...[/cyan]")
            console.print("[dim]Press Ctrl+C to stop[/dim]")
            
            # Wait for messages
            try:
                await asyncio.sleep(timeout)
                console.print(f"[yellow]Timeout reached. Received {received_count} messages.[/yellow]")
            except KeyboardInterrupt:
                console.print(f"\n[yellow]Stopped. Received {received_count} messages.[/yellow]")
            
        finally:
            await router.disconnect()
    
    asyncio.run(listen_messages())


@cli.command()
@click.argument("protocol", type=click.Choice([p.name.lower() for p in BrokerType]))
@click.option("--config", "-c", help="Configuration JSON string or file path")
def health(protocol: str, config: Optional[str]):
    """Check health of the specified protocol."""
    
    async def check_health():
        # Parse protocol
        broker_type = BrokerType[protocol.upper()]
        
        # Parse config
        config_dict = {}
        if config:
            try:
                if config.startswith("{"):
                    config_dict = json.loads(config)
                else:
                    with open(config, 'r') as f:
                        config_dict = json.load(f)
            except Exception as e:
                console.print(f"[red]Error parsing config: {e}[/red]")
                return
        
        # Deploy router
        try:
            router = await deploy_router(broker_type, config=config_dict)
            console.print(f"[green]Connected to {protocol.upper()} broker[/green]")
        except Exception as e:
            console.print(f"[red]Failed to connect: {e}[/red]")
            return
        
        try:
            # Get health information
            health_info = await router.health_check()
            
            console.print(Panel(
                JSON.from_data(health_info),
                title=f"{protocol.upper()} Health Check",
                border_style="green" if health_info.get("is_connected") else "red"
            ))
            
        finally:
            await router.disconnect()
    
    asyncio.run(check_health())


@cli.command()
@click.option("--interval", "-i", default=5, help="Update interval in seconds")
def monitor():
    """Monitor HybridPipe metrics in real-time."""
    
    def create_metrics_table():
        metrics_collector = get_metrics_collector()
        metrics = metrics_collector.get_metrics_summary()
        
        table = Table(title="HybridPipe Metrics")
        table.add_column("Metric", style="cyan")
        table.add_column("Value", style="green")
        
        table.add_row("Uptime", f"{metrics.get('uptime_seconds', 0):.1f}s")
        table.add_row("Total Messages", str(metrics.get('total_messages', 0)))
        table.add_row("Total Errors", str(metrics.get('total_errors', 0)))
        table.add_row("Message Rate", f"{metrics.get('message_rate_per_second', 0):.2f}/s")
        table.add_row("Error Rate", f"{metrics.get('error_rate', 0):.2%}")
        
        return table
    
    console.print("[cyan]Starting HybridPipe monitor...[/cyan]")
    console.print("[dim]Press Ctrl+C to stop[/dim]")
    
    try:
        with Live(create_metrics_table(), refresh_per_second=1/interval) as live:
            while True:
                live.update(create_metrics_table())
                asyncio.sleep(interval)
    except KeyboardInterrupt:
        console.print("\n[yellow]Monitor stopped[/yellow]")


@cli.command()
@click.argument("source_protocol", type=click.Choice([p.name.lower() for p in BrokerType]))
@click.argument("dest_protocol", type=click.Choice([p.name.lower() for p in BrokerType]))
@click.option("--source-config", help="Source configuration JSON")
@click.option("--dest-config", help="Destination configuration JSON")
@click.option("--source-pipe", default="source.channel", help="Source pipe name")
@click.option("--dest-pipe", default="dest.channel", help="Destination pipe name")
def bridge(source_protocol: str, dest_protocol: str, source_config: Optional[str], 
          dest_config: Optional[str], source_pipe: str, dest_pipe: str):
    """Bridge messages between two protocols."""
    
    async def bridge_messages():
        # Parse protocols
        source_broker = BrokerType[source_protocol.upper()]
        dest_broker = BrokerType[dest_protocol.upper()]
        
        # Parse configs
        source_config_dict = {}
        dest_config_dict = {}
        
        if source_config:
            try:
                source_config_dict = json.loads(source_config)
            except json.JSONDecodeError as e:
                console.print(f"[red]Error parsing source config: {e}[/red]")
                return
        
        if dest_config:
            try:
                dest_config_dict = json.loads(dest_config)
            except json.JSONDecodeError as e:
                console.print(f"[red]Error parsing dest config: {e}[/red]")
                return
        
        # Deploy routers
        try:
            source_router = await deploy_router(source_broker, config=source_config_dict)
            dest_router = await deploy_router(dest_broker, config=dest_config_dict)
            console.print(f"[green]Bridge established: {source_protocol.upper()} -> {dest_protocol.upper()}[/green]")
        except Exception as e:
            console.print(f"[red]Failed to establish bridge: {e}[/red]")
            return
        
        message_count = 0
        
        async def bridge_handler(data: bytes, metadata: MessageMetadata):
            nonlocal message_count
            
            from hybridpipe.serialization.engine import decode
            message = decode(data)
            
            # Forward message to destination
            await dest_router.dispatch(dest_pipe, message)
            message_count += 1
            
            if message_count % 100 == 0:
                console.print(f"[cyan]Bridged {message_count} messages[/cyan]")
        
        try:
            # Subscribe to source
            await source_router.subscribe(source_pipe, bridge_handler)
            console.print(f"[cyan]Bridging from '{source_pipe}' to '{dest_pipe}'[/cyan]")
            console.print("[dim]Press Ctrl+C to stop[/dim]")
            
            # Keep running
            try:
                while True:
                    await asyncio.sleep(1)
            except KeyboardInterrupt:
                console.print(f"\n[yellow]Bridge stopped. Processed {message_count} messages.[/yellow]")
            
        finally:
            await source_router.disconnect()
            await dest_router.disconnect()
    
    asyncio.run(bridge_messages())


def main():
    """Main CLI entry point."""
    try:
        cli()
    except KeyboardInterrupt:
        console.print("\n[yellow]Operation cancelled[/yellow]")
        sys.exit(1)
    except Exception as e:
        console.print(f"[red]Error: {e}[/red]")
        sys.exit(1)


if __name__ == "__main__":
    main()
