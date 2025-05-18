const std = @import("std");

pub fn main() !void {
    const src = [_]u8{1, 2, 3};
    var dest = [_]u8{0, 0, 0};
    
    // Try different memory copy methods
    std.mem.copyForwards(u8, &dest, &src);
    std.debug.print("After copyForwards: {any}\n", .{dest});
    
    dest = [_]u8{0, 0, 0};
    std.mem.copyBackwards(u8, &dest, &src);
    std.debug.print("After copyBackwards: {any}\n", .{dest});
    
    dest = [_]u8{0, 0, 0};
    @memcpy(&dest, &src);
    std.debug.print("After @memcpy: {any}\n", .{dest});
}
