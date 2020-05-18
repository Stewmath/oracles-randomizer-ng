local addrs = nil
local seasons_addrs = {
	multiPlayerNumber = 0x3f25,
	wGameState = 0xc2ee,
	wNetCountIn = 0xc6a1,
	wNetTreasureIn = 0xcbfb,
	wNetPlayerOut = 0xcbfd,
	wNetTreasureOut = 0xcbfe,
	wActiveGroup = 0xcc49,
	wActiveRoom = 0xcc4c,
}
local ages_addrs = {
	multiPlayerNumber = 0x3f1b,
	wGameState = 0xc2ee,
	wNetCountIn = 0xc6a9,
	wNetTreasureIn = 0xcbfb,
	wNetPlayerOut = 0xcbfd,
	wNetTreasureOut = 0xcbfe,
	wActiveGroup = 0xcc2d,
	wActiveRoom = 0xcc30,
}

-- converts a return value from memory.readbyterange to a string
local function string_from_byterange(br)
	local t = {}
	for k, v in pairs(br) do
		t[tonumber(k) + 1] = tonumber(v)
	end
	return string.char(unpack(t))
end

-- returns true iff func returns true for any element of a list
local function any_element_matches(list, func)
	for _, v in ipairs(list) do
		if func(v) then return true end
	end
	return false
end

-- figure out whether we're playing seasons or ages (or neither)
local game_code = string_from_byterange(memory.readbyterange(0x134, 9))
if game_code == "ZELDA DIN" then
	addrs = seasons_addrs
elseif game_code == "ZELDA NAY" then
	addrs = ages_addrs
else
	error("unknown ROM")
end

local this_player = memory.readbyte(addrs.multiPlayerNumber)
local items_in = {}
local items_out = {}
local oracles_ram = {} -- exports RAM controller interface

-- Gets a message to send to the other player of new changes
-- Returns the message as a dictionary object
-- Returns false if no message is to be sent
function oracles_ram.getMessage()
	-- return false if the player isn't in-game
	if memory.readbyte(addrs.wGameState) ~= 2 then
		return false
	end

	local count_in = memory.readbyte(addrs.wNetCountIn)
	if #items_in > count_in then
		-- give the most recent item to the game every frame until
		-- counts match
		local item = items_in[count_in + 1]
		memory.writebyte(addrs.wNetTreasureIn, item.id)
		memory.writebyte(addrs.wNetTreasureIn + 1, item.param)
	elseif #items_in < count_in then
		-- something is wrong if the save file's item count is higher
		-- than the RAM controller's, likely a disconnect. reset it so
		-- that the player can resync. TODO: items sent while a player
		-- is not connected will currently be lost forever
		console.log("resetting item count")
		memory.writebyte(addrs.wNetCountIn, #items_in)
	end

	local message = {}

	-- buffered treasure out? send and clear it
	local out_player = memory.readbyte(addrs.wNetPlayerOut)
	if out_player ~= 0 then
		-- get and clear vars
		local out_id = memory.readbyte(addrs.wNetTreasureOut)
		local out_param = memory.readbyte(addrs.wNetTreasureOut + 1)
		memory.writebyte(addrs.wNetPlayerOut, 0)
		memory.writebyte(addrs.wNetTreasureOut, 0)
		memory.writebyte(addrs.wNetTreasureOut + 1, 0)

		-- send message if room's item hasn't been sent before
		local room = memory.readbyte(addrs.wActiveGroup) * 0x100 +
			memory.readbyte(addrs.wActiveRoom)
		if any_element_matches(items_out, function(e)
			return e.room == room
		end) then
			console.log(string.format("item from P%d:%04x already sent",
				this_player, room))
		else
			rooms_out[room] = true
			message["m"] = {
				from = this_player,
				to = out_player,
				id = out_id,
				param = out_param,
				room = room,
			}
			table.insert(items_out, message["m"])
			console.log(string.format("sent item to P%d: {%02x, %02x}",
				out_player, out_id, out_param))
		end
	end

	-- return the message if it has content
	for _, __ in pairs(message) do return message end
	return false
end

-- Process a message from another player and update RAM
function oracles_ram.processMessage(their_user, message)
	if message["m"] ~= nil then
		local item = message["m"]
		if item.to == this_player then
			if any_element_matches(items_in, function(e)
				return e.player == item.player and e.room == item.room
			end) then
				console.log(string.format("item from P%d:%04x already received",
					item.from, item.room))
			else
				table.insert(items_in, item)
				console.log(string.format("received item from P%d: {%02x, %02x}",
					item.from, item.id, item.param))
			end
		end
	end
end

oracles_ram.itemcount = 1 -- dummy value, must be a positive integer

return oracles_ram
